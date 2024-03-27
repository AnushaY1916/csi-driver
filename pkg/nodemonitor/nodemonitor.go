// Copyright 2020 Hewlett Packard Enterprise Development LP

package nodemonitor

import (
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/tunelinux"
	"github.com/hpe-storage/csi-driver/pkg/flavor"
)

const (
	defaultIntervalSec = 30
	minimumIntervalSec = 15
)

func NewNodeMonitor(flavor flavor.Flavor, monitorInterval int64) *NodeMonitor {
	nm := &NodeMonitor{flavor: flavor, intervalSec: monitorInterval}
	if key := os.Getenv("NODE_NAME"); key != "" {
		nm.nodeName = key
	}
	log.Infof("NODE MONITOR: %+v", nm)
	// initialize node monitor
	return nm
}

// Monitor Pods running on un-reachable nodes
type NodeMonitor struct {
	flavor      flavor.Flavor
	intervalSec int64
	lock        sync.Mutex
	started     bool
	stopChannel chan int
	done        chan int
	nodeName    string
}

// StartMonitor starts the monitor
func (nm *NodeMonitor) StartNodeMonitor() error {
	log.Trace(">>>>> StartNodeMonitor")
	defer log.Trace("<<<<< StartNodeMonitor")

	nm.lock.Lock()
	defer nm.lock.Unlock()

	if nm.started {
		return fmt.Errorf("Node monitor has already been started")
	}

	if nm.intervalSec == 0 {
		nm.intervalSec = defaultIntervalSec
	} else if nm.intervalSec < minimumIntervalSec {
		log.Warnf("minimum interval for health monitor is %v seconds", minimumIntervalSec)
		nm.intervalSec = minimumIntervalSec
	}

	nm.stopChannel = make(chan int)
	nm.done = make(chan int)

	if err := nm.monitorNode(); err != nil {
		return err
	}

	nm.started = true
	return nil
}

// StopMonitor stops the monitor
func (nm *NodeMonitor) StopNodeMonitor() error {
	log.Trace(">>>>> StopNodeMonitor")
	defer log.Trace("<<<<< StopNodeMonitor")

	nm.lock.Lock()
	defer nm.lock.Unlock()

	if !nm.started {
		return fmt.Errorf("Node monitor has not been started")
	}

	close(nm.stopChannel)
	<-nm.done

	nm.started = false
	return nil
}

func (nm *NodeMonitor) monitorNode() error {
	log.Trace(">>>>> monitorNode")
	defer log.Trace("<<<<< monitorNode")
	defer close(nm.done)

	tick := time.NewTicker(time.Duration(nm.intervalSec) * time.Second)

	go func() {
		for {
			select {
			case <-tick.C:
				log.Infof("Node monitor started monitoring the node %s", nm.nodeName)
				multipathDevices, err := tunelinux.GetMultipathDevices() //driver.GetMultipathDevices()
				if err != nil {
					log.Errorf("Error while getting the multipath devices on the node %s", nm.nodeName)
					return
				}
				if multipathDevices != nil && len(multipathDevices) > 0 {
					for _, device := range multipathDevices {
						//TODO: Assess whether the device belongs to this node or not and whether to do clean up or not
						log.Tracef("Name:%s Vendor:%s Paths:%f Path Faults:%f UUID:%s IsUnhealthy:%t", device.Name, device.Vendor, device.Paths, device.PathFaults, device.UUID, device.IsUnhealthy)
						//Remove Later
						if device.IsUnhealthy {
							log.Infof("Multipath device %s is unhealthy and is present on %s node", device.Name, nm.nodeName)
						} else {
							log.Infof("Multipath device %s is healthy and is present on %s node", device.Name, nm.nodeName)
						}
					}
				} else {
					log.Tracef("No multipath devices found on the node %s", nm.nodeName)
				}
			case <-nm.stopChannel:
				return
			}
		}
	}()
	return nil
}
