// Copyright (c) 2017 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Binary simple-client is an example VPP management application that exercises the
// govpp API on real-world use-cases.
package vppbridge

// Generates Go bindings for all VPP APIs located in the json directory.
//go:generate binapi-generator --input-dir=../../bin_api --output-dir=../../bin_api

import (
	"fmt"

	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core/bin_api/l2"
)


//
// Constants
//
const debugBridge = false


//
// API Functions
//


// Check whether generated API messages are compatible with the version
// of VPP which the library is connected to.
func BridgeCompatibilityCheck(ch *api.Channel) error {
	err := ch.CheckMessageCompatibility(
		&l2.BridgeDomainAddDel{},
		&l2.BridgeDomainAddDelReply{},
		&l2.BridgeDomainDump{},
		&l2.BridgeDomainDetails{},
		&l2.SwInterfaceSetL2Bridge{},
		&l2.SwInterfaceSetL2BridgeReply{},
	)
	if err != nil {
		if debugBridge {
			fmt.Println("VPP memif failed compatibility")
		}
	}

	return err
}


// Attempt to create a Bridge Domain.
func CreateBridge(ch *api.Channel, bridgeDomain uint32) (error) {

	exists,_ := findBridge(ch, bridgeDomain)
	if exists {
		if debugBridge {
			fmt.Printf("Bridge Domain %d already exist, exit\n", bridgeDomain)
		}
		return nil
	}

	// Populate the Request Structure
	req := &l2.BridgeDomainAddDel{
		BdID: bridgeDomain,
		Flood: 1,
		UuFlood: 1,
		Forward: 1,
		Learn: 1,
		ArpTerm: 0,
		MacAge: 0,
		//BdTag   []byte `struc:"[64]byte"`
		IsAdd: 1,
	}

	reply := &l2.BridgeDomainAddDelReply{}

	err := ch.SendRequest(req).ReceiveReply(reply)

	if err != nil {
		if debugBridge {
			fmt.Println("Error creating bridge domain:", err)
		}
		return err
	}

	return err
}


// Attempt to delete a Bridge Domain.
func DeleteBridge(ch *api.Channel, bridgeDomain uint32) error {

	// Determine if bridge domain exists
	exists,count := findBridge(ch, bridgeDomain)
	if exists == false || count != 0 {
		return nil
	}

	// Populate the Request Structure
	req := &l2.BridgeDomainAddDel{
		BdID: bridgeDomain,
		IsAdd: 0,
	}

	reply := &l2.BridgeDomainAddDelReply{}

	err := ch.SendRequest(req).ReceiveReply(reply)

	if err != nil {
		if debugBridge {
			fmt.Println("Error deleting Bridge Domain:", err)
		}
		return err
	}

	return err
}


// Attempt to add an interface to a Bridge Domain.
func AddBridgeInterface(ch *api.Channel, bridgeDomain uint32, swIfId uint32) (error) {
	var err error

	// Determine if bridge domain exists, and if not, create it. CreateBridge()
	// checks for existance.
	err = CreateBridge(ch, bridgeDomain)
	if err != nil {
		return err
	}

	// Populate the Request Structure
	req := &l2.SwInterfaceSetL2Bridge{
		BdID: bridgeDomain,
		RxSwIfIndex: swIfId,
		Shg: 0,
		Bvi: 0,
		Enable: 1,
	}

	reply := &l2.SwInterfaceSetL2BridgeReply{}

	err = ch.SendRequest(req).ReceiveReply(reply)

	if err != nil {
		if debugBridge {
			fmt.Println("Error adding interface to bridge domain:", err)
		}
		return err
	}

	return err
}


// Attempt to remove an interface from a Bridge Domain.
func RemoveBridgeInterface(ch *api.Channel, bridgeDomain uint32, swIfId uint32) error {

	// Populate the Request Structure
	req := &l2.SwInterfaceSetL2Bridge{
		BdID: bridgeDomain,
		RxSwIfIndex: swIfId,
		Shg: 0,
		Bvi: 0,
		Enable: 0,
	}

	reply := &l2.SwInterfaceSetL2BridgeReply{}

	err := ch.SendRequest(req).ReceiveReply(reply)

	if err != nil {
		if debugBridge {
			fmt.Println("Error removing interface from bridge domain:", err)
		}
		return err
	}

	// DeleteBridge() checks to see if there are any interfaces still attached,
	// and if so, bail. So attempt to delete and let it validate.
	err = DeleteBridge(ch, bridgeDomain)
	if err != nil {
		return err
	}


	return err
}


// Dump the input Bridge data to Stdout. There is not VPP API to dump
// all the Bridges. 
func DumpBridge(ch *api.Channel, bridgeDomain uint32) {

        // Populate the Message Structure
	req := &l2.BridgeDomainDump{
		BdID: bridgeDomain,
	}

	reply := &l2.BridgeDomainDetails{}

	err := ch.SendRequest(req).ReceiveReply(reply)

	if err == nil {
		fmt.Printf("    Bridge Domain %d: Fld=%d UuFld=%d Fwd=%d Lrn=%d Arp=%d Mac=%d Bvi=%d NSwId=%d BdTag=%s\n",
			bridgeDomain,
			reply.Flood,
			reply.UuFlood,
			reply.Forward,
			reply.Learn,
			reply.ArpTerm,
			reply.MacAge,
			reply.BviSwIfIndex,
			reply.NSwIfs,
			string(reply.BdTag))

		if reply.NSwIfs != 0 {
			for i := uint32(0); i < reply.NSwIfs; i++ {
				fmt.Printf("      SwId=%d Shg=%d\n",
					reply.SwIfDetails[i].SwIfIndex,
					reply.SwIfDetails[i].Shg)
			}
		}
	} else {
		fmt.Printf("Bridge Domain %d does NOT Exist.\n", bridgeDomain)
	}
}


//
// Local Functions
//

// Determine if the input Bridge exists.
// Return: true - Exists  false - otherwise
//         uint32 - Number of associated interfaces
func findBridge(ch *api.Channel, bridgeDomain uint32) (bool,uint32) {
	var rval bool = false
	var count uint32

        // Populate the Message Structure
	req := &l2.BridgeDomainDump{
		BdID: bridgeDomain,
	}
	reqCtx := ch.SendMultiRequest(req)

	// BridgeDomainDump only returns one message, but if the Bridge Domain
	// doesn't exist, no response is returned and Reply times out. So use
	// SendMultiRequest to handle possible no response.
	for {
		reply := &l2.BridgeDomainDetails{}
		stop, err := reqCtx.ReceiveReply(reply)
		if stop {
			if debugBridge {
				fmt.Printf("Bridge Domain %d does NOT exist\n", bridgeDomain)
			}
			break // break out of the loop
		} else if err != nil {
			if debugBridge {
				fmt.Printf("Error searching for Bridge Domain %d\n", bridgeDomain)
			}
			break // break out of the loop
		} else {
			count = reply.NSwIfs
		}

		rval = true
	}

	return rval,count
}


