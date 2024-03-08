package initcontainer

import (
	"os"

	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/tunelinux"
	"github.com/hpe-storage/csi-driver/pkg/flavor"
	"github.com/hpe-storage/csi-driver/pkg/flavor/kubernetes"
	"github.com/hpe-storage/csi-driver/pkg/flavor/vanilla"
)

type InitContainer struct {
	flavor   flavor.Flavor
	nodeName string
}

func NewInitContainer(flavorName string, nodeService bool) *InitContainer {
	var initFlavour flavor.Flavor
	if flavorName == flavor.Kubernetes {
		flavor, err := kubernetes.NewKubernetesFlavor(nodeService, nil)
		if err != nil {
			return nil
		}
		initFlavour = flavor
	} else {
		initFlavour = &vanilla.Flavor{}
	}
	ic := &InitContainer{flavor: initFlavour}
	if key := os.Getenv("NODE_NAME"); key != "" {
		ic.nodeName = key
	}
	log.Infof("InitContainer: %+v", ic)
	// initialize InitContainer
	return ic
}
func (ic *InitContainer) Init() error {

	log.Trace(">>>>> init method of Init Container")
	defer log.Trace("<<<<< init method of INit Container")

	multipathDevices, err := tunelinux.GetMultipathDevices() //driver.GetMultipathDevices()
	if err != nil {
		log.Errorf("Error while getting the multipath devices")
		return err
	}
	unhealthyDevices, err := tunelinux.GetUnhealthyMultipathDevices(multipathDevices)
	if err != nil {
		log.Errorf("Error while retreiving unhealthy devices: %s", err.Error())
	}
	log.Tracef("Unhealthy devices found are: %+v", unhealthyDevices)

	if len(unhealthyDevices) > 0 {
		log.Tracef("Unhealthy devices found on the node %s", ic.nodeName)
		vaList, err := ic.flavor.ListVolumeAttachments()
		if err != nil {
			return err
		}
		if len(vaList.Items) > 0 {
			log.Infof("Volume Attachments are more")
			for _, va := range vaList.Items {
				log.Info("Volume Attachment: ", va, &va)
				log.Info("NAME:", va.Name)
				log.Info("PV:", va.Spec.Source.PersistentVolumeName)
				log.Info("NODE NAME:", va.Spec.NodeName)
			}
		}

	} else {
		log.Tracef("No unhealthy devices found on teh node %s", ic.nodeName)
	}
	return nil
}
