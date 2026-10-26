package message

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	schemaBaseURL       = "https://schemas.collector.example/"
	schemaSuffix        = ".schema.json"
	contractVersionMark = "v"
)

var errNoContract = errors.New("aucun contrat de message trouvé")

type contractKey struct {
	messageType string
	version     int64
}

type catalog map[contractKey]*jsonschema.Schema

func loadCatalog(contracts fs.FS) (catalog, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	contractPaths, err := registerSchemas(compiler, contracts)
	if err != nil {
		return nil, err
	}
	if len(contractPaths) == 0 {
		return nil, errNoContract
	}
	return compileContracts(compiler, contractPaths)
}

func registerSchemas(compiler *jsonschema.Compiler, contracts fs.FS) (map[contractKey]string, error) {
	contractPaths := make(map[contractKey]string)
	err := fs.WalkDir(contracts, ".", func(schemaPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(schemaPath, schemaSuffix) {
			return nil
		}
		if err := addSchema(compiler, contracts, schemaPath); err != nil {
			return err
		}
		if key, isContract := contractKeyOf(schemaPath); isContract {
			contractPaths[key] = schemaPath
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lecture des contrats de messages : %w", err)
	}
	return contractPaths, nil
}

func addSchema(compiler *jsonschema.Compiler, contracts fs.FS, schemaPath string) error {
	content, err := fs.ReadFile(contracts, schemaPath)
	if err != nil {
		return fmt.Errorf("lecture du schéma %s : %w", schemaPath, err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("analyse du schéma %s : %w", schemaPath, err)
	}
	if err := compiler.AddResource(schemaURL(schemaPath), document); err != nil {
		return fmt.Errorf("enregistrement du schéma %s : %w", schemaPath, err)
	}
	return nil
}

func compileContracts(compiler *jsonschema.Compiler, contractPaths map[contractKey]string) (catalog, error) {
	compiled := make(catalog, len(contractPaths))
	for key, schemaPath := range contractPaths {
		schema, err := compiler.Compile(schemaURL(schemaPath))
		if err != nil {
			return nil, fmt.Errorf("compilation du schéma %s : %w", schemaPath, err)
		}
		compiled[key] = schema
	}
	return compiled, nil
}

func contractKeyOf(schemaPath string) (contractKey, bool) {
	digits, hasMark := strings.CutPrefix(path.Base(schemaPath), contractVersionMark)
	if !hasMark {
		return contractKey{}, false
	}
	digits, hasSuffix := strings.CutSuffix(digits, schemaSuffix)
	if !hasSuffix {
		return contractKey{}, false
	}
	version, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || version < 1 {
		return contractKey{}, false
	}
	return contractKey{messageType: path.Base(path.Dir(schemaPath)), version: version}, true
}

func schemaURL(schemaPath string) string {
	return schemaBaseURL + schemaPath
}
