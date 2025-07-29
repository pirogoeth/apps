package config

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/sirupsen/logrus"
)

// ApplyDefaults iterates through the fields on a struct, checking if the
// current value is a zero value. If so, and the field has a "default" struct tag,
// sets that into the value.
func ApplyDefaults(item any) error {
	container := reflect.ValueOf(item)
	containerKind := container.Kind()
	if containerKind == reflect.Pointer {
		// If item is a pointer to a struct, dereference
		if container.IsNil() {
			return fmt.Errorf("`item` must not be nil!")
		}
		container = container.Elem()
	} else {
		// Otherwise, make a new value of the same type and copy the data
		ptr := reflect.New(container.Type())
		ptr.Elem().Set(container)
		container = ptr.Elem()
	}

	// container = reflect.Indirect(container)
	containerType := container.Type()
	for idx := range containerType.NumField() {
		field := container.Field(idx)
		structField := containerType.Field(idx)

		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				// Initialize the underlying struct
				field.Set(reflect.New(field.Type().Elem()))
				logrus.Tracef("initialized underlying type %v for %v", field.Type(), field)
			}

			field = field.Elem()
		}

		logrus.Tracef("ApplyDefaults: %v.%v (type %v): field kind = %v",
			container.Type().Name(), structField.Name, structField.Type,
			field.Kind(),
		)

		// Apply defaults recursively
		if field.Kind() == reflect.Struct {
			if err := ApplyDefaults(field.Addr().Interface()); err != nil {
				return err
			}

			continue
		} else if field.Kind() == reflect.Array || field.Kind() == reflect.Slice {
			// TODO: Making a big assumption here that arrays and slices are treated equally in reflect
			logrus.Tracef("Field %v is array or slice: %#v", structField.Name, field)
			logrus.Tracef("Field type elem is %v", field.Type().Elem().Kind())
			logrus.Tracef("Field type kind is %v", field.Type().Kind())

			// Only recurse if array/slice element type is a struct
			innerFieldTypeKind := field.Type().Elem().Kind()
			if innerFieldTypeKind != reflect.Struct {
				continue
			}

			logrus.Tracef("applying defaults on array/slice items %v", field)
			for innerFieldItemIdx := range field.Len() {
				if err := ApplyDefaults(field.Index(innerFieldItemIdx).Addr().Interface()); err != nil {
					return err
				}
			}
		}

		if !field.IsZero() {
			continue
		}

		fieldDefault, hasDefault := structField.Tag.Lookup("default")
		if !hasDefault {
			continue
		}

		if err := setFieldDefault(container, field, structField, fieldDefault); err != nil {
			return err
		}
	}

	return nil
}

func setFieldDefault(container reflect.Value, field reflect.Value, structField reflect.StructField, fieldDefault string) error {
	// Check for pointers - initialize the value if empty
	logrus.Tracef("setFieldDefault(%#v, %#v, %#v, %#v)\n", container, field, structField, fieldDefault)

	switch field.Kind() {
	case reflect.Struct:
		// Recurse into nested structs
		if err := ApplyDefaults(field.Addr().Interface()); err != nil {
			nestedTypeName := reflect.ValueOf(field.Interface()).Type().Name()
			return fmt.Errorf("error applying defaults on nested struct '%v.%v' (%v): %w", container.Type().Name(), structField.Name, nestedTypeName, err)
		}
	case reflect.Bool:
		defaultVal, err := strconv.ParseBool(fieldDefault)
		if err != nil {
			return fmt.Errorf("error setting default on '%v.%v': %w", container.Type().Name(), structField.Name, err)
		}

		field.SetBool(defaultVal)
	case reflect.String:
		field.SetString(fieldDefault)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		defaultVal, err := strconv.ParseInt(fieldDefault, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("error setting default on '%v.%v': %w", container.Type().Name(), structField.Name, err)
		}

		field.SetInt(defaultVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		defaultVal, err := strconv.ParseUint(fieldDefault, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("error setting default on '%v.%v': %w", container.Type().Name(), structField.Name, err)
		}

		field.SetUint(defaultVal)
	case reflect.Float32, reflect.Float64:
		defaultVal, err := strconv.ParseFloat(fieldDefault, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("error setting default on '%v.%v': %w", container.Type().Name(), structField.Name, err)
		}

		field.SetFloat(defaultVal)
	case reflect.Complex64, reflect.Complex128:
		defaultVal, err := strconv.ParseComplex(fieldDefault, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("error setting default on '%v.%v': %w", container.Type().Name(), structField.Name, err)
		}

		container.SetComplex(defaultVal)
	}

	return nil
}
