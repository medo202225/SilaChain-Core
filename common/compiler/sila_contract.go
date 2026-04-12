// Copyright (c) 2026 SilaChain
// All rights reserved.
// Proprietary and confidential.
// Use of this source code is governed by the SilaChain license.

package compiler

import (
	"encoding/json"
	"fmt"
)

type silaCompilerOutputLegacy struct {
	Contracts map[string]struct {
		BinRuntime                                  string `json:"bin-runtime"`
		SrcMapRuntime                               string `json:"srcmap-runtime"`
		Bin, SrcMap, Abi, Devdoc, Userdoc, Metadata string
		Hashes                                      map[string]string
	}
	Version string
}

type silaCompilerOutputModern struct {
	Contracts map[string]struct {
		BinRuntime            string `json:"bin-runtime"`
		SrcMapRuntime         string `json:"srcmap-runtime"`
		Bin, SrcMap, Metadata string
		Abi                   interface{}
		Devdoc                interface{}
		Userdoc               interface{}
		Hashes                map[string]string
	}
	Version string
}

// ParseCombinedCompilerJSON parses combined compiler JSON output into SilaContract values.
func ParseCombinedCompilerJSON(
	combinedJSON []byte,
	source string,
	language string,
	languageVersion string,
	compilerVersion string,
	compilerOptions string,
) (map[string]*SilaContract, error) {
	var output silaCompilerOutputLegacy
	if err := json.Unmarshal(combinedJSON, &output); err != nil {
		return parseCombinedCompilerJSONModern(
			combinedJSON,
			source,
			language,
			languageVersion,
			compilerVersion,
			compilerOptions,
		)
	}

	contracts := make(map[string]*SilaContract)
	for name, info := range output.Contracts {
		var abi interface{}
		var userDoc interface{}
		var developerDoc interface{}

		if err := json.Unmarshal([]byte(info.Abi), &abi); err != nil {
			return nil, fmt.Errorf("compiler: error reading abi definition (%v)", err)
		}
		if err := json.Unmarshal([]byte(info.Userdoc), &userDoc); err != nil {
			return nil, fmt.Errorf("compiler: error reading userdoc definition (%v)", err)
		}
		if err := json.Unmarshal([]byte(info.Devdoc), &developerDoc); err != nil {
			return nil, fmt.Errorf("compiler: error reading devdoc definition (%v)", err)
		}

		contracts[name] = &SilaContract{
			Code:        "0x" + info.Bin,
			RuntimeCode: "0x" + info.BinRuntime,
			Hashes:      info.Hashes,
			Info: SilaContractInfo{
				Source:          source,
				Language:        language,
				LanguageVersion: languageVersion,
				CompilerVersion: compilerVersion,
				CompilerOptions: compilerOptions,
				SrcMap:          info.SrcMap,
				SrcMapRuntime:   info.SrcMapRuntime,
				AbiDefinition:   abi,
				UserDoc:         userDoc,
				DeveloperDoc:    developerDoc,
				Metadata:        info.Metadata,
			},
		}
	}
	return contracts, nil
}

func parseCombinedCompilerJSONModern(
	combinedJSON []byte,
	source string,
	language string,
	languageVersion string,
	compilerVersion string,
	compilerOptions string,
) (map[string]*SilaContract, error) {
	var output silaCompilerOutputModern
	if err := json.Unmarshal(combinedJSON, &output); err != nil {
		return nil, err
	}

	contracts := make(map[string]*SilaContract)
	for name, info := range output.Contracts {
		contracts[name] = &SilaContract{
			Code:        "0x" + info.Bin,
			RuntimeCode: "0x" + info.BinRuntime,
			Hashes:      info.Hashes,
			Info: SilaContractInfo{
				Source:          source,
				Language:        language,
				LanguageVersion: languageVersion,
				CompilerVersion: compilerVersion,
				CompilerOptions: compilerOptions,
				SrcMap:          info.SrcMap,
				SrcMapRuntime:   info.SrcMapRuntime,
				AbiDefinition:   info.Abi,
				UserDoc:         info.Userdoc,
				DeveloperDoc:    info.Devdoc,
				Metadata:        info.Metadata,
			},
		}
	}
	return contracts, nil
}
