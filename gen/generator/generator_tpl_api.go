package generator

import (
	"fmt"
	"os"
	"strings"

	"github.com/muka/go-bluetooth/gen"
	"github.com/muka/go-bluetooth/gen/override"
	log "github.com/sirupsen/logrus"
)

// getRawType clean tag from type
func getRawType(t string) string {
	if strings.Contains(t, "`") {
		p1 := strings.Trim(strings.Split(t, "`")[0], " ")
		return p1
	}
	return t
}

// getRawTypeInitializer return field initializer
func getRawTypeInitializer(t string) string {
	t = getRawType(t)

	// array
	if len(t) >= 2 && t[:2] == "[]" {
		return t + "{}"
	}
	// map
	if len(t) >= 3 && t[:3] == "map" {
		return t + "{}"
	}
	// int*
	if len(t) >= 3 && t[:3] == "int" {
		return t + "(0)"
	}
	// uint*
	if len(t) >= 4 && t[:4] == "uint" {
		return t + "(0)"
	}
	// float*
	if len(t) >= 5 && t[:5] == "float" {
		return t + "(0.0)"
	}

	switch t {
	case "bool":
		return "false"
	case "string":
		return "\"\""
	case "byte":
		return "[]uint8{}"
	case "dbus.ObjectPath":
		return "dbus.ObjectPath(\"\")"
	default:
		panic(fmt.Sprintf("Unknown type: %s", t))
	}
}

func ApiTemplate(filename string, api gen.Api, apiGroup gen.ApiGroup) error {

	fw, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %s", err)
	}

	apiName := getApiPackage(apiGroup)

	imports := []string{
		"sync",
		"github.com/muka/go-bluetooth/bluez",
	}

	// Expose Properties interface ?
	exposeProps := override.ExposeProperties(api.Interface)

	if exposeProps {
		imports = append(imports, "reflect")
		imports = append(imports, "github.com/fatih/structs")
		imports = append(imports, "github.com/muka/go-bluetooth/util")
	}

	// flag to import dbus
	// NOTE: set to true to handle dbus.Signalling
	// importDbus := false

	importDbus := true

	pts := strings.Split(api.Interface, ".")
	iface := pts[len(pts)-1]

	propsList := map[string]*gen.PropertyDoc{}

	for _, p := range api.Properties {

		prop := gen.PropertyDoc{
			Property: p,
		}

		prop.Name = strings.Trim(p.Name, ": \t")
		prop.Property.Docs = prepareDocs(p.Docs, true, 2)
		prop.Property.Type = castType(p.Type)
		prop.RawType = getRawType(prop.Property.Type)
		prop.RawTypeInitializer = getRawTypeInitializer(prop.Property.Type)
		propsList[prop.Name] = &prop
	}

	propertiesOverride, found := override.GetPropertiesOverride(api.Interface)
	if found {
		for propName, propType := range propertiesOverride {

			var prop *gen.PropertyDoc
			if _, ok := propsList[propName]; ok {
				prop = propsList[propName]
				prop.RawType = getRawType(prop.Property.Type)
				prop.RawTypeInitializer = getRawTypeInitializer(prop.Property.Type)
				prop.Property.Type = propType
				// log.Debugf("props --> %s %s", propName, propType)
			} else {
				prop = &gen.PropertyDoc{
					Property: gen.Property{
						Name: propName,
						Type: propType,
					},
					RawType:            getRawType(propType),
					RawTypeInitializer: getRawTypeInitializer(propType),
				}
				propsList[propName] = prop
			}

			if !importDbus {
				importDbus = strings.Contains(prop.Property.Type, "dbus.")
			}

		}
	}

	props := []gen.PropertyDoc{}
	for _, prop := range propsList {
		props = append(props, *prop)
	}

	methods := []gen.MethodDoc{}
	for _, m := range api.Methods {

		args := []string{}
		params := []string{}
		for _, a := range m.Args {
			arg := a.Name + " " + castType(a.Type)
			args = append(args, arg)
			params = append(params, a.Name)
		}

		mm := gen.MethodDoc{
			Method:     m,
			ArgsList:   strings.Join(args, ", "),
			ParamsList: strings.Join(params, ", "),
		}

		if !importDbus {
			importDbus = strings.Contains(mm.ArgsList, "dbus.")
			log.Debugf("%t %s", importDbus, mm.ArgsList)
		}
		if !importDbus {
			importDbus = strings.Contains(mm.ParamsList, "dbus.")
		}

		mm.Method.Name = strings.Replace(mm.Method.Name, " (optional)", "", -1)
		mm.Method.Docs = prepareDocs(mm.Method.Docs, true, 0)
		mm.Method.ReturnType = castType(mm.Method.ReturnType)

		mm.SingleReturn = len(mm.Method.ReturnType) == 0

		if mm.SingleReturn {
			mm.Method.ReturnType = "error"
		} else {

			// log.Debugf("With return type %s", mm.Method.ReturnType)

			returnTypes := strings.Split(mm.Method.ReturnType, ", ")
			defs := []string{}
			refs := []string{}
			list := []string{}
			for i, returnType := range returnTypes {

				// objInitialization1 := ""
				// objInitialization2 := ""
				// if strings.Contains(returnType, "[]") {
				// 	objInitialization1 = "="
				// 	objInitialization2 = "{}"
				// }

				varName := fmt.Sprintf("val%d", i)
				// def := fmt.Sprintf("var %s %s %s%s", varName, objInitialization1, returnType, objInitialization2)
				def := fmt.Sprintf("var %s %s", varName, returnType)
				ref := "&" + varName

				defs = append(defs, def)
				refs = append(refs, ref)
				list = append(list, varName)

			}

			mm.ReturnVarsDefinition = strings.Join(defs, "\n  ")
			mm.ReturnVarsRefs = strings.Join(refs, ", ")
			mm.ReturnVarsList = strings.Join(list, ", ")

			if !importDbus {
				importDbus = strings.Contains(mm.Method.ReturnType, "dbus.")
			}

			mm.Method.ReturnType = "(" + mm.Method.ReturnType + ", error)"
		}

		if len(mm.Method.Name) == 0 {
			continue
		}

		methods = append(methods, mm)
	}

	if importDbus {
		imports = append(imports, "github.com/godbus/dbus")
	}

	api.Description = prepareDocs(api.Description, false, 0)
	api.Title = strings.Trim(api.Title, "\n \t")

	ctrs := createConstructors(api)

	for _, c := range ctrs {
		importFmt := strings.Contains(c.ObjectPath, "fmt.")
		if !importFmt {
			importFmt = strings.Contains(c.Service, "fmt.")
		}
		if importFmt {
			imports = append(imports, "fmt")
		}
	}

	importsTpl := ""
	if len(imports) > 0 {
		for i := range imports {
			imports[i] = fmt.Sprintf(`"%s"`, imports[i])
		}
		importsTpl = fmt.Sprintf("import (\n  %s\n)", strings.Join(imports, "\n  "))
	}

	apidocs := gen.ApiDoc{
		Imports:          importsTpl,
		Package:          apiName,
		Api:              api,
		InterfaceName:    iface,
		Properties:       props,
		Methods:          methods,
		Constructors:     ctrs,
		ExposeProperties: exposeProps,
	}

	tmpl := loadtpl("api")
	err = tmpl.Execute(fw, apidocs)
	if err != nil {
		return fmt.Errorf("api tpl: %s", err)
	}

	// log.Debugf("Created %s", filename)
	return nil
}