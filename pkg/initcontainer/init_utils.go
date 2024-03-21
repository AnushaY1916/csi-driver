package initcontainer

import (
	"os"

	log "github.com/hpe-storage/common-host-libs/logger"
	"github.com/hpe-storage/common-host-libs/model"
	"github.com/hpe-storage/common-host-libs/tunelinux"
	"github.com/hpe-storage/csi-driver/pkg/flavor"
	"github.com/hpe-storage/csi-driver/pkg/flavor/kubernetes"
	"github.com/hpe-storage/csi-driver/pkg/flavor/vanilla"
	storage_v1 "k8s.io/api/storage/v1"
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
	defer log.Trace("<<<<< init method of Init Container")

	multipathDevices, err := tunelinux.GetMultipathDevices() //driver.GetMultipathDevices()
	if err != nil {
		log.Errorf("Error while getting the multipath devices")
		return err
	}
	if multipathDevices != nil && len(multipathDevices) > 0 {
		vaList, err := ic.flavor.ListVolumeAttachments()
		if err != nil {
			return
		}
		for _, device := range multipathDevices {
			if vaList != nil && len(vaList.Items) > 0 {
				log.Infof("Assessing the multipath device %s", device.Name)
				if doesDeviceBelongToTheNode(device, vaList, ic.nodeName) {
					log.Infof("Multipath device %s belongs to the node %s", device.Name, ic.nodeName)
					if device.IsUnhealthy {
						log.Info("The multipath device %s belongs to this node %s and is unhealthy. Issue warnings!", device.Name, ic.nodeName)
					} else {
						log.Info("The multipath device %s belongs to this node %s and is healthy. Nothing to do", device.Name, ic.nodeName)
					}
				} else {
					log.Infof("Multipath device %s doesnt not belong to the node %s", device.Name, ic.nodeName)
					if device.IsUnhealthy {
						log.Infof("The multipath device %s is unhealthy and it does not belong to the node %s", device.Name, ic.nodeName)
						//do cleanup
					} else {
						log.Infof("The multipath device %s is healthy and it does not belong to the node %s. Issue warnings!", device.Name, ic.nodeName)
					}
				}
			} else {
				if device.IsUnhealthy {
					log.Infof("No volume attachments found. The multipath device is unhealthy and does not belong to HPE CSI driver, Do cleanup!")
					// Do cleanup
				} else {
					log.Infof("No volume attachmenst found. The multipath device is healthy and does not belong to HPE CSI driver")
					//Nothing to do
				}
			}
		}
	} else {
		log.Infof("No multipath devices found on the node %s", ic.nodeName)
	}
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
