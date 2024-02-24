package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Datastore interface {
	SaveEnrollment(context.Context, Enrollment) error
	SaveDevice(context.Context, Device) error
	GetDevice(context.Context, string) (*Device, error)

	EnqueueCommands(context.Context, map[string][]byte, []*Device) error
	GetNextCommand(context.Context, string) ([]byte, error)
	SaveCommandResult(context.Context, string, string, []byte) error

	Close() error
}

type BoltDatastore struct {
	db *bolt.DB
}

func NewBoltStore(name string) (*BoltDatastore, error) {
	db, err := bolt.Open(name, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		buckets := []string{"enrollments", "devices", "command_queue", "commands", "command_results"}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("create bucket %s: %s", bucket, err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &BoltDatastore{db: db}, nil
}

func (b *BoltDatastore) Close() error {
	return b.db.Close()
}

func (b *BoltDatastore) SaveCommandResult(ctx context.Context, deviceUUID, commandUUID string, result []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		crb := tx.Bucket([]byte("command_results"))
		cqb := tx.Bucket([]byte("command_queue"))
		// TODO: check if a command exists and was enqueued before
		// allowing this
		key := []byte(deviceUUID + "-" + commandUUID)
		cqb.Delete(key)
		return crb.Put(key, result)
	})
}

func (b *BoltDatastore) EnqueueCommands(ctx context.Context, commands map[string][]byte, devices []*Device) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		cb := tx.Bucket([]byte("commands"))
		qb := tx.Bucket([]byte("command_queue"))

		// TODO: check that a command with the given UUID doesn't
		// already exist.
		for uuid, command := range commands {
			if err := cb.Put([]byte(uuid), command); err != nil {
				return err
			}

			for _, d := range devices {
				if err := qb.Put([]byte(d.UDID+"-"+uuid), []byte(uuid)); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (b *BoltDatastore) GetNextCommand(ctx context.Context, deviceUDID string) ([]byte, error) {
	var rawCmd []byte
	b.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("command_queue")).Cursor()

		prefix := []byte(deviceUDID)
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			b := tx.Bucket([]byte("commands"))
			if rawCmd = b.Get(v); rawCmd == nil {
				return errors.New("couldn't find enqueued command")
			}
			break
		}
		return nil
	})

	return rawCmd, nil
}

func (b *BoltDatastore) SaveEnrollment(ctx context.Context, enrollment Enrollment) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("enrollments"))
		bytes, err := json.Marshal(enrollment)
		if err != nil {
			return fmt.Errorf("marshalling enrollment: %w", err)
		}
		return b.Put([]byte(enrollment.EnrollmentID), bytes)
	})
}

func (b *BoltDatastore) GetDevice(ctx context.Context, udid string) (*Device, error) {
	raw, err := b.get("devices", udid)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := json.Unmarshal(raw, &device); err != nil {
		return nil, err
	}

	return &device, nil
}

func (b *BoltDatastore) get(bucket, key string) ([]byte, error) {
	var raw []byte
	b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		raw = bkt.Get([]byte(key))
		return nil
	})

	if raw == nil {
		return nil, errors.New("not found")
	}

	return raw, nil
}

func (b *BoltDatastore) SaveDevice(ctx context.Context, device Device) error {
	fmt.Println("save device called")
	return b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("devices"))
		bytes, err := json.Marshal(device)
		if err != nil {
			return fmt.Errorf("marshalling device: %w", err)
		}
		fmt.Printf("saving %s : %s\n", device.UDID, bytes)
		return b.Put([]byte(device.UDID), bytes)
	})
}

func (b *BoltDatastore) ListDevicesRaw() ([]byte, error) {
	barr := [][]byte{}
	b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("devices"))
		b.ForEach(func(k, v []byte) error {
			barr = append(barr, v)
			return nil
		})
		return nil
	})
	return bytes.Join([][]byte{{'['}, bytes.Join(barr, []byte{','}), {']'}}, []byte{' '}), nil
}
