package queue

import (
	"encoding/json"

	"github.com/RTradeLtd/Temporal/rtfs"

	"github.com/RTradeLtd/Temporal/models"
	"github.com/RTradeLtd/Temporal/tns"
	"github.com/RTradeLtd/config"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func (qm *QueueManager) ProcessTNSRecordCreation(msgs <-chan amqp.Delivery, db *gorm.DB, cfg *config.TemporalConfig) error {
	zm := models.NewZoneManager(db)
	rm := models.NewRecordManager(db)
	qm.Logger.WithFields(log.Fields{
		"service": qm.Service,
	}).Info("processing messages")
	for d := range msgs {
		qm.Logger.WithFields(log.Fields{
			"service": qm.Service,
		}).Info("new message received")
		req := RecordCreation{}
		if err := json.Unmarshal(d.Body, &req); err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to unmarshal message")
			d.Ack(false)
			continue
		}
		if _, err := zm.FindZoneByNameAndUser(req.ZoneName, req.UserName); err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("unable to find zone")
			d.Ack(false)
			continue
		}
		if _, err := zm.AddRecordForZone(
			req.ZoneName, req.RecordName, req.UserName,
		); err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("unable to add record to zone")
			d.Ack(false)
			continue
		}
		if _, err := rm.FindRecordByNameAndUser(req.RecordName, req.UserName); err == nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("record already exists in database")
			d.Ack(false)
			continue
		}
		if _, err := rm.AddRecord(
			req.UserName, req.RecordName, req.RecordKeyName, req.ZoneName, req.MetaData,
		); err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("unable to add record to database")
			d.Ack(false)
			continue
		}
		//TODO: add calls here that store the data to IPFS as an IPLD object
		qm.Logger.WithFields(log.Fields{
			"service": qm.Service,
		}).Info("record added to zone")
		d.Ack(false)
	}
	return nil
}

func (qm *QueueManager) ProcessTNSZoneCreation(msgs <-chan amqp.Delivery, db *gorm.DB, cfg *config.TemporalConfig) error {
	zm := models.NewZoneManager(db)
	qm.Logger.WithFields(log.Fields{
		"service": qm.Service,
	}).Info("processing messages")
	for d := range msgs {
		qm.Logger.WithFields(log.Fields{
			"service": qm.Service,
		}).Info("new message received")
		req := ZoneCreation{}
		if err := json.Unmarshal(d.Body, &req); err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to unmarshal message")
			d.Ack(false)
			continue
		}
		zone, err := zm.FindZoneByNameAndUser(req.Name, req.UserName)
		if err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to search for zone")
			d.Ack(false)
			continue
		}
		rtfsManager, err := rtfs.Initialize("", "")
		if err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to intiialize connection to ipfs")
			d.Ack(false)
			continue
		}
		if err = rtfsManager.CreateKeystoreManager(); err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to initialize keystore manager")
			d.Ack(false)
			continue
		}
		zoneManagerPK, err := rtfsManager.KeystoreManager.GetPrivateKeyByName(req.ManagerKeyName)
		if err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to get zone manager private key")
			d.Ack(false)
			continue
		}
		zonePK, err := rtfsManager.KeystoreManager.GetPrivateKeyByName(req.ZoneKeyName)
		if err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to initialize keystore manager")
			d.Ack(false)
			continue
		}
		z := tns.Zone{
			PublicKey: zonePK.GetPublic(),
			Manager: &tns.ZoneManager{
				PublicKey: zoneManagerPK.GetPublic(),
			},
			Name: req.Name,
		}
		z.PublicKey = zonePK.GetPublic()
		z.Manager = &tns.ZoneManager{
			PublicKey: zoneManagerPK.GetPublic(),
		}
		z.Name = req.Name
		marshaled, err := json.Marshal(&z)
		if err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to marshaled tns zone")
			d.Ack(false)
			continue
		}
		resp, err := rtfsManager.Shell.DagPut(marshaled, "json", "cbor")
		if err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to put zone file to ipfs")
			d.Ack(false)
			continue
		}
		zone.LatestIPFSHash = resp
		if _, err = zm.UpdateLatestIPFSHashForZone(zone.Name, zone.UserName, resp); err != nil {
			qm.Logger.WithFields(log.Fields{
				"service": qm.Service,
				"error":   err.Error(),
			}).Error("failed to update zone in database")
			d.Ack(false)
			continue
		}
		qm.Logger.WithFields(log.Fields{
			"service": qm.Service,
		}).Info("zone published and database is updated")
		d.Ack(false)
		continue
	}
	return nil
}
