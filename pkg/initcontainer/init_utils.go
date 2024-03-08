package initcontainer

import (
	"os"

	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/tunelinux"
	"github.com/hpe-storage/csi-driver/pkg/flavor"
)

func init() err {

	log.Trace(">>>>> init method of Init Container")
	defer log.Trace("<<<<< init method of INit Container")
	var nodeName string
	if key := os.Getenv("NODE_NAME"); key != "" {
		nodeName = key
	}
	multipathDevices, err := tunelinux.GetMultipathDevices() //driver.GetMultipathDevices()
	if err != nil {
		log.Errorf("Error while getting the multipath devices")
		return
	}
	unhealthyDevices, err := tunelinux.GetUnhealthyMultipathDevices(multipathDevices)
	if err != nil {
		log.Errorf("Error while retreiving unhealthy devices: %s", err.Error())
	}
	log.Tracef("Unhealthy devices found are: %+v", unhealthyDevices)

	if len(unhealthyDevices) > 0 {
		log.Tracef("Unhealthy devices found on the node %s", nodeName)
		vaList, err := flavor.ListVolumeAttachments()
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
		log.Tracef("No unhealthy devices found on teh node %s", nodeName)
	}
}
