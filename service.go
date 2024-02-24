package main

import (
	"context"
	"fmt"
)

type Service struct {
	ds        Datastore
	commander Commander
}

func (s *Service) Authenticate(ctx context.Context, enrollment Enrollment, auth Authenticate) error {
	fmt.Printf("Authenticate called: %+v, %+v\n", enrollment, auth)
	device := Device{
		UDID:         enrollment.UDID,
		BuildVersion: auth.BuildVersion,
		Name:         auth.DeviceName,
		IMEI:         auth.IMEI,
		MEID:         auth.MEID,
		Model:        auth.Model,
		ModelName:    auth.ModelName,
		OSVersion:    auth.OSVersion,
		ProductName:  auth.ProductName,
		SerialNumber: auth.SerialNumber,
	}
	if err := s.ds.SaveDevice(ctx, device); err != nil {
		return err
	}
	return nil
}

func (s *Service) TokenUpdate(ctx context.Context, enrollment Enrollment, auth TokenUpdate) error {
	device, err := s.ds.GetDevice(ctx, enrollment.UDID)
	if err != nil {
		return err
	}
	device.PushMagic = auth.PushMagic
	device.Token = auth.Token
	device.UnlockToken = auth.UnlockToken
	if err := s.ds.SaveDevice(ctx, *device); err != nil {
		return err
	}
	return nil
}

func (s *Service) SetBootstrapToken(ctx context.Context, enrollment Enrollment, auth SetBootstrapToken) error {
	device, err := s.ds.GetDevice(ctx, enrollment.UDID)
	if err != nil {
		return err
	}
	device.BootstrapToken = auth.BootstrapToken
	if err := s.ds.SaveDevice(ctx, *device); err != nil {
		return err
	}
	return nil
}

func (s *Service) CheckOut(ctx context.Context, enrollment Enrollment) error {
	fmt.Println("CheckOut called", enrollment)
	return nil
}

func (s *Service) DeclarativeManagement(ctx context.Context, enrollment Enrollment, mgmt DeclarativeManagement) error {
	fmt.Println("DeclarativeManagement called", enrollment)
	return nil
}

func (s *Service) CommandsHandler(ctx context.Context, enrollment Enrollment, cmd CommandResult, raw []byte) ([]byte, error) {
	fmt.Println("CommandsHandler called", enrollment, cmd)
	switch cmd.Status {
	case "Idle":
		return s.ds.GetNextCommand(ctx, enrollment.UDID)
	case "Acknowledged", "Error":
		if err := s.ds.SaveCommandResult(ctx, enrollment.UDID, cmd.CommandUUID, raw); err != nil {
			return nil, fmt.Errorf("saving command result: %w", err)
		}
		return s.ds.GetNextCommand(ctx, enrollment.UDID)
	case "NotNow":
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected command status: %s", cmd.Status)
	}
}
