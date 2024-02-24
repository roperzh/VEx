package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/groob/plist"
)

type Commander struct {
	ds         Datastore
	apnsClient *Client
}

func (c *Commander) Enqueue(
	ctx context.Context,
	devices []*Device,
	commands map[string][]byte,
) error {
	if err := c.ds.EnqueueCommands(ctx, commands, devices); err != nil {
		return err
	}

	notifications := make([]*Notification, len(devices)*len(commands))
	for i, d := range devices {
		notifications[i] = &Notification{
			Token:     hex.EncodeToString(d.Token),
			PushMagic: d.PushMagic,
		}
	}

	// TODO: loop results
	c.apnsClient.Push(ctx, notifications)
	return nil
}

type Command struct {
	RequestType string
	Data        []byte
}

type CommandWrapper struct {
	CommandUUID string
	Command
}

func (c *Commander) DeclarativeManagement(
	ctx context.Context,
	data DeclarativeManagementData,
	devices []*Device,
) error {
	rawData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	cmd := CommandWrapper{
		CommandUUID: uuid.NewString(),
		Command: Command{
			RequestType: "DeclarativeManagement",
			Data:        rawData,
		},
	}

	cmdBytes, err := plist.MarshalIndent(cmd, "  ")
	if err != nil {
		return err
	}

	fmt.Println(cmdBytes)
	//
	//	cmd := fmt.Sprintf(`
	//<?xml version="1.0" encoding="UTF-8"?>
	//<!DOCTYPE plist PUBLIC “-//Apple//DTD PLIST 1.0//EN" “http://www.apple.com/DTDs/PropertyList-1.0.dtd">
	//<plist version="1.0">
	//<dict>
	//    <key>Command</key>
	//    <dict>
	//        <key>CommandUUID</key>
	//        <string>%s</string>
	//        <key>Command</key>
	//        <dict>
	//            <key>RequestType</key>
	//            <string>DeclarativeManagement</string>
	//            <key>Data</key>
	//            <data>%s</data>
	//        </dict>
	//    </dict>
	//</dict>
	//</plist>`, cmdUUID, data64)

	return c.Enqueue(ctx, devices, map[string][]byte{cmd.CommandUUID: []byte(cmdBytes)})
}
