// Copyright 2022 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package objstore

import (
	"encoding/json"
	"io"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/platform/api"
)

var objStoreInsertCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new object of a given type",
	Long: `This command allows the creation of a new object of a given type in the Object Store.

	Usage:
	fsoc objstore create --type<fully-qualified-typename> --object-file=<fully-qualified-path> --layer-type=<valid-layer-type> [--layer-id=<valid-layer-id>]
	
	Flags/Options:
	--type - Flag to indicate the fully qualified type name of the object that you would like to create
	--object-file - Flag to indicate the fully qualified path (from your root directory) to the file containing the definition of the object that you want to create
	--layer-type - Flag to indicate the layer at which you would like to create your object
	--layer-id - OPTIONAL Flag to specify a custom layer ID for the object that you would like to create.  This is calculated automatically for all layers currently supported but can be overridden with this flag`,

	Args:             cobra.ExactArgs(0),
	Run:              insertObject,
	TraverseChildren: true,
}

func getCreateObjectCmd() *cobra.Command {
	objStoreInsertCmd.Flags().
		String("type", "", "The fully qualified type name of the object")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("type")

	objStoreInsertCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the object definition")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("objectFile")

	objStoreInsertCmd.Flags().
		String("layer-type", "", "The layer-type that the created object will be added to")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("layer-type")

	objStoreInsertCmd.Flags().
		String("layer-id", "", "The layer-id that the created object will be added to. Optional for TENANT and SOLUTION layers ")

	return objStoreInsertCmd

}

func insertObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Errorf("Can't find the object definition file named %s", objJsonFilePath)
		return
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)
	var objectStruct map[string]interface{}
	err = json.Unmarshal(objectBytes, &objectStruct)
	if err != nil {
		log.Errorf("Can't generate a %s object from the %s file. Make sure the object definition has all the required field and is valid according to the type definition.")
		return
	}

	layerType, _ := cmd.Flags().GetString("layer-type")
	layerID := getCorrectLayerID(layerType, objType)

	if layerID == "" {
		if !cmd.Flags().Changed("layer-id") {
			log.Error("Unable to set layer-id flag from given context. Please specify a unique layer-id value with the --layer-id flag")
			return
		}
		layerID, err = cmd.Flags().GetString("layer-id")
		if err != nil {
			log.Errorf("error trying to get %q flag value: %w", "layer-id", err)
			return
		}
	}

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   layerID,
	}

	var res any
	// objJsonStr, err := json.Marshal(objectStruct)
	err = api.JSONPost(getObjStoreObjectUrl()+"/"+objType, objectStruct, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Errorf("objstore command failed: %v", err.Error())
		return
	} else {
		log.Infof("Successfully created %s object", objType)
	}
}

func getObjStoreObjectUrl() string {
	return "objstore/v1beta/objects"
}

var objStoreInsertPatchedObjectCmd = &cobra.Command{
	Use:   "create-patch",
	Short: "Create a new patched object of a given type",
	Long: `This command allows the creation of a new patched object of a given type in the Object Store.
	A patched object inherits values from an object that exists at a higher layer and can also override mutable fields when needed.


	Usage:
	fsoc objstore create-patch --type<fully-qualified-typename> --object-file=<fully-qualified-path> --target-layer-type=<valid-layer-type> --parent-object-id=<valid-object-id>`,

	Args:             cobra.ExactArgs(0),
	Run:              insertPatchObject,
	TraverseChildren: true,
}

func getCreatePatchObjectCmd() *cobra.Command {
	objStoreInsertPatchedObjectCmd.Flags().
		String("type", "", "The fully qualified type name of the object")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("type")

	objStoreInsertPatchedObjectCmd.Flags().
		String("parent-object-id", "", "The id of the parent object for which you want to create a patched object at a lower layer")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("parent-object-id")

	objStoreInsertPatchedObjectCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the object definition")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("objectFile")

	objStoreInsertPatchedObjectCmd.Flags().
		String("target-layer-type", "", "The layer-type at which the patch object will be created. For inheritance purposes, this should always be a `lower` layer than the parent object's layer")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("target-layer-type")

	return objStoreInsertPatchedObjectCmd
}

func insertPatchObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")
	parentObjId, _ := cmd.Flags().GetString("parent-object-id")

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Errorf("Can't find the object definition file named %s", objJsonFilePath)
		return
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)
	var objectStruct map[string]interface{}
	err = json.Unmarshal(objectBytes, &objectStruct)
	if err != nil {
		log.Errorf("Can't generate a %s object from the %s file. Make sure the object definition has all the required fields and is valid according to the type definition.")
		return
	}

	layerType, _ := cmd.Flags().GetString("target-layer-type")
	layerID := getCorrectLayerID(layerType, objType)

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   layerID,
	}

	var res any
	err = api.JSONPatch(getObjStoreObjectUrl()+"/"+objType+"/"+parentObjId, objectStruct, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Errorf("Creating a patched object command failed: %v", err.Error())
		return
	} else {
		log.Infof("Successfully created patched %s object at the %s layer", objType, layerType)
	}
}
