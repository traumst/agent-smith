package main
import (
	"fmt"
	"reflect"
	"google.golang.org/api/serviceusage/v1"
)
func main() {
	var svc serviceusage.Service
	typ := reflect.TypeOf(svc)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fmt.Printf("Field: %s, Type: %s\n", field.Name, field.Type)
		// If it's a pointer to a service, check its fields too
		if field.Type.Kind() == reflect.Ptr {
			subTyp := field.Type.Elem()
			if subTyp.Kind() == reflect.Struct {
				for j := 0; j < subTyp.NumField(); j++ {
					subField := subTyp.Field(j)
					fmt.Printf("  SubField: %s.%s, Type: %s\n", field.Name, subField.Name, subField.Type)
					if subField.Type.Kind() == reflect.Ptr {
						subSubTyp := subField.Type.Elem()
						if subSubTyp.Kind() == reflect.Struct {
							for k := 0; k < subSubTyp.NumField(); k++ {
								subSubField := subSubTyp.Field(k)
								fmt.Printf("    SubSubField: %s.%s.%s, Type: %s\n", field.Name, subField.Name, subSubField.Name, subSubField.Type)
							}
						}
					}
				}
			}
		}
	}
}
