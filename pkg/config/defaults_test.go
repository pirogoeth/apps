package config_test

import (
	"os"
	"testing"

	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logging.Setup()
	logrus.SetFormatter(new(logrus.TextFormatter))
	os.Exit(m.Run())
}

type Example struct {
	A int   `default:"1"`
	B int64 `default:"5"`

	C uint   `default:"10"`
	D uint64 `default:"14"`

	E float32 `default:"20.002"`
	F float64 `default:"42.0000000420"`

	G string `default:"yearly"`
}

func TestApplyDefaults(t *testing.T) {
	item := new(Example)
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if item.A != 1 {
		t.Errorf("Example.A unexpected value %v", item.A)
	}

	if item.B != 5 {
		t.Errorf("Example.B unexpected value %v", item.B)
	}

	if item.C != 10 {
		t.Errorf("Example.C unexpected value %v", item.C)
	}

	if item.D != 14 {
		t.Errorf("Example.D unexpected value %v", item.D)
	}

	if item.E != 20.002 {
		t.Errorf("Example.E unexpected value %v", item.E)
	}

	if item.F != 42.0000000420 {
		t.Errorf("Example.F unexpected value %v", item.F)
	}

	if item.G != "yearly" {
		t.Errorf("Example.G unexpected value %v", item.G)
	}
}

type ExampleWithNestedStruct struct {
	Something string `default:"is decidedly insane"`

	NestedAnon struct {
		A int `default:"29"`
	}
}

type ExampleWithNestedPointerUninitialized struct {
	Something string `default:"has a sane default"`

	NestedAnon *struct {
		A int `default:"1"`
	}
}

func TestApplyDefaultsNestedPointerUninitialized(t *testing.T) {
	item := new(ExampleWithNestedPointerUninitialized)
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if item.Something != "has a sane default" {
		t.Errorf("ExampleWithNestedPointerUninitialized.Something unexpected value %v", item.Something)
	}

	if item.NestedAnon.A != 1 {
		t.Errorf("ExampleWithNestedPointerUninitialized.Nested.A unexpected value %v", item.NestedAnon.A)
	}
}

func TestApplyDefaultsNestedPointerInitialized(t *testing.T) {
	item := new(ExampleWithNestedPointerUninitialized)
	item.NestedAnon = new(struct {
		A int `default:"1"`
	})
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if item.Something != "has a sane default" {
		t.Errorf("ExampleWithNestedPointerUninitialized.Something unexpected value %v", item.Something)
	}

	if item.NestedAnon.A != 1 {
		t.Errorf("ExampleWithNestedPointerUninitialized.Nested.A unexpected value %v", item.NestedAnon.A)
	}
}

func TestApplyDefaultsNestedPointerInitializedDoesNotOverwrite(t *testing.T) {
	item := new(ExampleWithNestedPointerUninitialized)
	item.NestedAnon = new(struct {
		A int `default:"1"`
	})
	item.NestedAnon.A = 42
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if item.Something != "has a sane default" {
		t.Errorf("ExampleWithNestedPointerUninitialized.Something unexpected value %v", item.Something)
	}

	if item.NestedAnon.A != 42 {
		t.Errorf("ExampleWithNestedPointerUninitialized.Nested.A unexpected value %v", item.NestedAnon.A)
	}
}

type MyOtherItem struct {
	B int64 `default:"42"`
}

type ExampleWithExternalStruct struct {
	Something string `default:"yet another sane default"`

	Thing *MyOtherItem
}

func TestApplyDefaultsExternalStructInitialized(t *testing.T) {
	item := new(ExampleWithExternalStruct)
	item.Thing = new(MyOtherItem)
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if item.Something != "yet another sane default" {
		t.Errorf("ExampleWithExternalStruct.Something unexpected value %v", item.Something)
	}

	if item.Thing.B != 42 {
		t.Errorf("ExampleWithExternalStruct.Thing.A unexpected value %v", item.Thing.B)
	}
}

func TestApplyDefaultsExternalStructUninitialized(t *testing.T) {
	item := new(ExampleWithExternalStruct)
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if item.Something != "yet another sane default" {
		t.Errorf("ExampleWithExternalStruct.Something unexpected value %v", item.Something)
	}

	if item.Thing.B != 42 {
		t.Errorf("ExampleWithExternalStruct.Thing.A unexpected value %v", item.Thing.B)
	}
}

type ExampleWithTypeMismatch struct {
	A int64 `default:"what do you mean I'm an int???"`
	B bool  `default:"no"`
}

func TestExampleWithTypeMismatch(t *testing.T) {
	item := new(ExampleWithTypeMismatch)
	if err := config.ApplyDefaults(item); err == nil {
		t.Errorf("expected `error setting default...: strconv.ParseInt`")
	}
}

type ExampleDetail struct {
	A string `default:"twenty"`
	B int
}

type ExampleWithArrayOfStruct struct {
	Details []ExampleDetail
}

func TestExampleWithArrayOfStructEmpty(t *testing.T) {
	item := new(ExampleWithArrayOfStruct)
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if len(item.Details) > 0 {
		t.Errorf("expected ExampleWithArrayOfStruct.Details to be empty")
	}
}

func TestExampleWithArrayOfStruct(t *testing.T) {
	item := new(ExampleWithArrayOfStruct)
	item.Details = append(item.Details, ExampleDetail{
		A: "am valued",
		B: 42,
	})
	item.Details = append(item.Details, ExampleDetail{
		B: 99,
	})
	if err := config.ApplyDefaults(item); err != nil {
		t.Errorf("test ApplyDefaults failed: %s", err.Error())
	}

	if val := item.Details[0].A; val != "am valued" {
		t.Errorf("ExampleWithArrayOfStruct[0].A has unexpected value %v", val)
	}

	if val := item.Details[0].B; val != 42 {
		t.Errorf("ExampleWithArrayOfStruct[0].B has unexpected value %v", val)
	}

	if val := item.Details[1].A; val != "twenty" {
		t.Errorf("ExampleWithArrayOfStruct[1].A has unexpected value %v", val)
	}

	if val := item.Details[1].B; val != 99 {
		t.Errorf("ExampleWithArrayOfStruct[1].B has unexpected value %v", val)
	}
}
