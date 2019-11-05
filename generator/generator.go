package generator

import (
	"errors"
	"github.com/victorolegovich/storage_generator/collection"
	"github.com/victorolegovich/storage_generator/parser"
	reg "github.com/victorolegovich/storage_generator/register"
	"github.com/victorolegovich/storage_generator/settings"
	_go "github.com/victorolegovich/storage_generator/templates/go"
	"github.com/victorolegovich/storage_generator/validator"
	"os"
)

const config = "/home/victor/go/src/github.com/victorolegovich/test/config/user.json"

type genSection int

const (
	parsing genSection = iota
	validating
	templating
	register
)

var gsToString = map[genSection]string{
	parsing:    "parsing",
	validating: "validating",
	templating: "templating",
	register:   "register",
}

func (gs genSection) string() string {
	return gsToString[gs]
}

type processError map[genSection]error

type Generator struct {
	settings settings.Settings
	processError
}

func (gen *Generator) Generate() error {
	s, e := settings.New(config)

	if e != nil {
		return e
	}

	gen.settings = s
	gen.processError = map[genSection]error{}

	c := &collection.Collection{}
	r, e := reg.NewRegister()
	if e != nil {
		gen.processError[register] = e
	}
	rObj := &reg.RegObject{}
	rObj.Entistor = map[string]string{}

	if err := parser.Parse(gen.settings.Path.DataDir, c); err != nil {
		gen.processError[parsing] = err
	} else {
		rObj.Package = c.DataPackage
	}

	if err := validator.StructsValidation(c.Structs); err != nil {
		gen.processError[validating] = err
	}

	template := _go.NewTemplate(*c, gen.settings)

	for _, file := range template.Create() {
		if _, err := os.Stat(file.Path); os.IsNotExist(err) {
			if err = os.Mkdir(file.Path, os.ModePerm); err != nil {
				gen.processError[templating] = err
			}
		}

		if f, err := os.Create(file.Path + "/" + file.Name); err == nil {
			rObj.Entistor[file.Owner] = file.Path

			if _, err = f.Write([]byte(file.Src)); err != nil {
				gen.processError[templating] = err
			}
		} else {
			gen.processError[templating] = err
		}
	}

	r.AddObject(*rObj)

	if err := r.Save(); err != nil {
		gen.processError[register] = err
	}

	if gen.settings.AutoDelete {
		for _, del := range r.Deleted {
			_ = os.RemoveAll(del)
		}
	}

	return errorConverting(gen.processError)
}

func errorConverting(pErr processError) error {
	var errorText string

	for section, err := range pErr {
		errorText += section.string() + " section of generating: \n\t" + err.Error() + "\n"
	}

	if errorText != "" {
		return errors.New(errorText)
	}

	return nil
}