// Copyright 2020 Hewlett Packard Enterprise Development LP

package nodemonitor

import (
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"github.com/hpe-storage/common-host-libs/tunelinux"
	"github.com/hpe-storage/csi-driver/pkg/flavor"
	storage_v1 "k8s.io/api/storage/v1"
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
				log.Infof("NODE MONITOR :Monitoring node......1")
				multipathDevices, err := tunelinux.GetMultipathDevices() //driver.GetMultipathDevices()
				if err != nil {
					log.Errorf("Error while getting the multipath devices")
					return
				}
				if multipathDevices != nil && len(multipathDevices) > 0 {
					unhealthyDevices, err := tunelinux.GetUnhealthyMultipathDevices(multipathDevices)
					if err != nil {
						log.Errorf("Error while retreiving unhealthy multipath devices: %s", err.Error())
					}
					log.Tracef("Unhealthy multipath devices found are: %+v", unhealthyDevices)
					vaList, err := nm.flavor.ListVolumeAttachments()
					if err != nil {
						return
					}
					if len(unhealthyDevices) > 0 {
						log.Tracef("Unhealthy devices found on the node %s", nm.nodeName)
						if vaList != nil && len(vaList.Items) > 0 {
							for _, device := range multipathDevices {
								if doesDeviceBelongToTheNode(device, vaList, nm.nodeName) {
									log.Info("The multipath device %s belongs to this node %s and is unhealthy. Issue warnings!", device.Name, nm.nodeName)
								} else {
									log.Infof("The multipath device %s is unhealthy and it does not belong to the node %s", device.Name, nm.nodeName)
									//do cleanup
								}
							}
						} else if len(vaList.Items) == 0 {
							log.Tracef("No volume attachments found. The multipath devices is unhealthy and does not belong to HPE CSI driver, Do cleanup!")
							// Do cleanup
						}
						//Do cleanup
					} else {
						log.Tracef("No unhealthy devices found on the node %s", nm.nodeName)
						//check whether they belong to this node or not

						if vaList != nil && len(vaList.Items) > 0 {
							for _, device := range multipathDevices {
								if doesDeviceBelongToTheNode(device, vaList, nm.nodeName) {
									log.Info("The multipath device %s belongs to this node %s and is healthy. Nothing to do", device.Name, nm.nodeName)
								} else {
									//do cleanup
									log.Infof("The multipath device %s is healthy and it does not belong to the node %s. Issue warnings!", device.Name, nm.nodeName)
								}
							}
						} else if len(vaList.Items) == 0 {
							log.Tracef("No volume attachmenst found. The multipath device is healthy and does not belong to HPE CSI driver")
						}
					}
				} else {
					log.Tracef("No multipath devices found on the node %s", nm.nodeName)
				}
				log.Infof("NODE MONITOR :Monitoring node......2")
			case <-nm.stopChannel:
				return
			}
		}
	}()
	return nil
}

func doesDeviceBelongToTheNode(multipathDevice *model.MultipathDeviceInfo, volumeAttachmentList *storage_v1.VolumeAttachmentList, nodeName string) bool {
	if multipathDevice != nil {
		for _, va := range volumeAttachmentList.Items {
			log.Info("NAME:", va.Name)
			log.Info("PV:", *va.Spec.Source.PersistentVolumeName)
			log.Info("STATUS: ", va.Status)
			log.Info("ATTATCHMENTMETADATA: ", va.Status.AttachmentMetadata)
			log.Info("SERIAL NUMBER: ", va.Status.AttachmentMetadata["serialNumber"])
			log.Info("NODE NAME:", va.Spec.NodeName)

			if multipathDevice.UUID[1:] == va.Status.AttachmentMetadata["serialNumber"] && nodeName == va.Spec.NodeName {
				return true
			}
		}
	}
	return false
}
