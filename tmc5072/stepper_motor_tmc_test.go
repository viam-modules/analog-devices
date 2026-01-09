//go:build linux

// Package tmc5072 contains the TMC stepper motor driver. This file contains unit tests for it.
package tmc5072

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go.viam.com/rdk/components/board/genericlinux/buses"
	"go.viam.com/rdk/components/motor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/testutils/inject"
	"go.viam.com/test"
)

type fakeSpiHandle struct {
	tx, rx [][]byte // tx and rx must have the same length
	i      int      // Index of the next tx/rx pair to use
	tb     testing.TB
}

func newFakeSpiHandle(tb testing.TB) fakeSpiHandle {
	h := fakeSpiHandle{}
	h.rx = [][]byte{}
	h.tx = [][]byte{}
	h.i = 0
	h.tb = tb
	return h
}

func (h *fakeSpiHandle) Xfer(
	ctx context.Context,
	baud uint,
	chipSelect string,
	mode uint,
	tx []byte,
) ([]byte, error) {
	test.That(h.tb, tx, test.ShouldResemble, h.tx[h.i])
	result := h.rx[h.i]
	h.i++
	return result, nil
}

func (h *fakeSpiHandle) Close() error {
	return nil
}

func (h *fakeSpiHandle) AddExpectedTx(expects [][]byte) {
	for _, line := range expects {
		h.tx = append(h.tx, line)
		h.rx = append(h.rx, make([]byte, len(line)))
	}
}

func (h *fakeSpiHandle) AddExpectedRx(expects, sends [][]byte) {
	h.tx = append(h.tx, expects...)
	h.rx = append(h.rx, sends...)
}

func (h *fakeSpiHandle) ExpectDone() {
	// Assert that all expected data was transmitted
	test.That(h.tb, h.i, test.ShouldEqual, len(h.tx))
}

func newFakeSpi(tb testing.TB) (*fakeSpiHandle, buses.SPI) {
	handle := newFakeSpiHandle(tb)
	fakeSpi := inject.SPI{}
	fakeSpi.OpenHandleFunc = func() (buses.SPIHandle, error) {
		return &handle, nil
	}

	return &handle, &fakeSpi
}

const maxRpm = 500

func TestRPMBounds(t *testing.T) {
	ctx := context.Background()
	logger, obs := logging.NewObservedTestLogger(t)

	getLastLogLine := func() string {
		lastLine := ""
		// Filter out the debug logs that just indicate what we sent/received on the SPI bus.
		for _, lineVal := range obs.All() {
			line := fmt.Sprint(lineVal)
			if strings.Contains(line, "Read from ") || strings.Contains(line, "Write to ") {
				continue
			}
			lastLine = line
		}
		return lastLine
	}

	fakeSpiHandle, fakeSpi := newFakeSpi(t)
	var deps resource.Dependencies

	mc := Config{
		SPIBus:           "3",
		ChipSelect:       "40",
		Index:            1,
		SGThresh:         0,
		CalFactor:        1.0,
		MaxAcceleration:  500,
		MaxRPM:           maxRpm,
		TicksPerRotation: 200,
	}

	// These are the setup register writes
	fakeSpiHandle.AddExpectedTx([][]byte{
		{236, 0, 1, 0, 195},
		{176, 0, 6, 15, 8},
		{237, 0, 0, 0, 0},
		{164, 0, 0, 21, 8},
		{166, 0, 0, 21, 8},
		{170, 0, 0, 21, 8},
		{168, 0, 0, 21, 8},
		{163, 0, 0, 0, 1},
		{171, 0, 0, 0, 10},
		{165, 0, 2, 17, 149},
		{177, 0, 0, 105, 234},
		{167, 0, 0, 0, 0},
		{160, 0, 0, 0, 1},
		{161, 0, 0, 0, 0},
	})

	name := resource.NewName(motor.API, "motor1")
	motorDep, err := makeMotor(ctx, deps, mc, name, logger, fakeSpi)
	test.That(t, err, test.ShouldBeNil)
	defer func() {
		fakeSpiHandle.ExpectDone()
		test.That(t, motorDep.Close(context.Background()), test.ShouldBeNil)
	}()

	test.That(t, motorDep.GoFor(ctx, 0.05, 6.6, nil), test.ShouldBeError, motor.NewZeroRPMError())
	test.That(t, getLastLogLine(), test.ShouldContainSubstring, "nearly 0")

	// Check with position at 0.0 revolutions
	//nolint:dupl
	fakeSpiHandle.AddExpectedRx(
		[][]byte{
			{33, 0, 0, 0, 0},
			{33, 0, 0, 0, 0},
			{160, 0, 0, 0, 0},
			{164, 0, 0, 21, 8},   // a1
			{166, 0, 0, 21, 8},   // aMax
			{170, 0, 0, 21, 8},   // d1
			{168, 0, 0, 21, 8},   // dMax
			{163, 0, 0, 0, 1},    // vStart
			{171, 0, 0, 0, 10},   // vStop
			{165, 0, 2, 17, 149}, // v1
			{167, 0, 8, 70, 85},
			{173, 0, 5, 40, 0},
			{53, 0, 0, 0, 0},
			{53, 0, 0, 0, 0},
		},
		[][]byte{
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 2, 0},
		},
	)
	test.That(t, motorDep.GoFor(ctx, 500, 6.6, nil), test.ShouldBeNil)
	test.That(t, getLastLogLine(), test.ShouldContainSubstring, "nearly the max")
}

func TestTMCStepperMotor(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger(t)

	fakeSpiHandle, fakeSpi := newFakeSpi(t)
	var deps resource.Dependencies

	mc := Config{
		SPIBus:           "main",
		ChipSelect:       "40",
		Index:            1,
		SGThresh:         0,
		CalFactor:        1.0,
		MaxAcceleration:  500,
		MaxRPM:           maxRpm,
		TicksPerRotation: 200,
	}

	// These are the setup register writes
	fakeSpiHandle.AddExpectedTx([][]byte{
		{236, 0, 1, 0, 195},
		{176, 0, 6, 15, 8},
		{237, 0, 0, 0, 0},
		{164, 0, 0, 21, 8},
		{166, 0, 0, 21, 8},
		{170, 0, 0, 21, 8},
		{168, 0, 0, 21, 8},
		{163, 0, 0, 0, 1},
		{171, 0, 0, 0, 10},
		{165, 0, 2, 17, 149},
		{177, 0, 0, 105, 234},
		{167, 0, 0, 0, 0},
		{160, 0, 0, 0, 1},
		{161, 0, 0, 0, 0},
	})

	name := resource.NewName(motor.API, "motor1")
	motorDep, err := makeMotor(ctx, deps, mc, name, logger, fakeSpi)
	test.That(t, err, test.ShouldBeNil)
	defer func() {
		fakeSpiHandle.ExpectDone()
		test.That(t, motorDep.Close(context.Background()), test.ShouldBeNil)
	}()

	t.Run("motor supports position reporting", func(t *testing.T) {
		properties, err := motorDep.Properties(ctx, nil)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, properties.PositionReporting, test.ShouldBeTrue)
	})

	t.Run("motor SetPower testing", func(t *testing.T) {
		// Test Go forward at half speed
		fakeSpiHandle.AddExpectedTx([][]byte{
			{160, 0, 0, 0, 1},
			{167, 0, 4, 35, 42},
		})
		test.That(t, motorDep.SetPower(ctx, 0.5, nil), test.ShouldBeNil)

		// Test Go backward at quarter speed
		fakeSpiHandle.AddExpectedTx([][]byte{
			{160, 0, 0, 0, 2},
			{167, 0, 2, 17, 149},
		})
		test.That(t, motorDep.SetPower(ctx, -0.25, nil), test.ShouldBeNil)
	})

	t.Run("motor SetRPM testing", func(t *testing.T) {
		// Test Go forward at half speed
		fakeSpiHandle.AddExpectedTx([][]byte{
			{160, 0, 0, 0, 1},    // rampMode
			{164, 0, 0, 21, 8},   // a1
			{166, 0, 0, 21, 8},   // aMax
			{170, 0, 0, 21, 8},   // d1
			{168, 0, 0, 21, 8},   // dMax
			{163, 0, 0, 0, 1},    // vStart
			{171, 0, 0, 0, 10},   // vStop
			{165, 0, 2, 17, 149}, // v1
			{167, 0, 4, 35, 42},  // vMax
		})
		test.That(t, motorDep.SetRPM(ctx, 250, nil), test.ShouldBeNil)

		fakeSpiHandle.AddExpectedTx([][]byte{
			{160, 0, 0, 0, 1},    // rampMode
			{164, 0, 0, 11, 184}, // a1
			{166, 0, 0, 13, 172}, // aMax
			{170, 0, 0, 9, 196},  // d1
			{168, 0, 0, 14, 16},  // dMax
			{163, 0, 0, 0, 0},    // vStart
			{171, 0, 0, 0, 10},   // vStop
			{165, 0, 0, 4, 232},  // v1
			{167, 0, 4, 35, 42},  // vMax
		})
		rampParams := map[string]any{
			"ramp_parameters": map[string]any{
				"v_start": 0,
				"v_stop":  10,
				"v1":      1256,
				"a1":      3000,
				"d1":      2500,
				"a_max":   3500,
				"d_max":   3600,
			},
		}
		test.That(t, motorDep.SetRPM(ctx, 250, rampParams), test.ShouldBeNil)

		// Test Go backward at quarter speed
		fakeSpiHandle.AddExpectedTx([][]byte{
			{160, 0, 0, 0, 2},    // rampMode
			{164, 0, 0, 21, 8},   // a1
			{166, 0, 0, 21, 8},   // aMax
			{170, 0, 0, 21, 8},   // d1
			{168, 0, 0, 21, 8},   // dMax
			{163, 0, 0, 0, 1},    // vStart
			{171, 0, 0, 0, 10},   // vStop
			{165, 0, 2, 17, 149}, // v1
			{167, 0, 2, 17, 149}, // vMax
		})
		test.That(t, motorDep.SetRPM(ctx, -125, nil), test.ShouldBeNil)
	})

	t.Run("motor Off testing", func(t *testing.T) {
		fakeSpiHandle.AddExpectedTx([][]byte{
			{160, 0, 0, 0, 1},
			{167, 0, 0, 0, 0},
		})
		test.That(t, motorDep.Stop(ctx, nil), test.ShouldBeNil)
	})

	t.Run("motor position testing", func(t *testing.T) {
		// Check at 4.0 revolutions
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 1, 6, 18, 3}, // Can be gibberish, only second register is valid
				{0, 0, 3, 32, 0},
			},
		)
		pos, err := motorDep.Position(ctx, nil)
		test.That(t, pos, test.ShouldEqual, 4.0)
		test.That(t, err, test.ShouldBeNil)
	})

	t.Run("motor GoTo with positive rpm and positive revolutions", func(t *testing.T) {
		// GoTo 3.2 at 50 rpm with default ramp parameters
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 2, 128, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoTo(ctx, 50.0, 3.2, nil), test.ShouldBeNil)

		// GoTo -3.2 at 50 rpm with custom ramp parameters
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{160, 0, 0, 0, 0},
				{164, 0, 0, 11, 184},  // a1
				{166, 0, 0, 13, 172},  // aMax
				{170, 0, 0, 9, 196},   // d1
				{168, 0, 0, 14, 16},   // dMax
				{163, 0, 0, 0, 0},     // vStart
				{171, 0, 0, 0, 10},    // vStop
				{165, 0, 0, 4, 232},   // v1
				{167, 0, 0, 211, 213}, // vMax
				{173, 255, 253, 128, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)

		rampParams := map[string]any{
			"ramp_parameters": map[string]any{
				"v_start": 0,
				"v_stop":  10,
				"v1":      1256,
				"a1":      3000,
				"d1":      2500,
				"a_max":   3500,
				"d_max":   3600,
			},
		}

		test.That(t, motorDep.GoTo(ctx, 50.0, -3.2, rampParams), test.ShouldBeNil)
		// GoTo 0 at 50 rpm with default ramp parameters
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoTo(ctx, 50.0, 0, nil), test.ShouldBeNil)
	})

	t.Run("motor GoFor with positive rpm and positive revolutions", func(t *testing.T) {
		// Check with position at 0.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 2, 128, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, 50.0, 3.2, nil), test.ShouldBeNil)

		// Check with position at 4.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 5, 160, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 8, 98, 98, 7}, // Can be gibberish
				{0, 0, 3, 32, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, 50.0, 3.2, nil), test.ShouldBeNil)

		// Check with position at 1.2 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 6, 24, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 240, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, 50.0, 6.6, nil), test.ShouldBeNil)
	})

	t.Run("motor GoFor with negative rpm and positive revolutions", func(t *testing.T) {
		// Check with position at 0.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 255, 253, 128, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, -50.0, 3.2, nil), test.ShouldBeNil)

		// Check with position at 4.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 0, 159, 255},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 8, 98, 98, 7}, // Can be gibberish
				{0, 0, 3, 32, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, -50.0, 3.2, nil), test.ShouldBeNil)

		// Check with position at 1.2 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 255, 251, 200, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 240, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, -50.0, 6.6, nil), test.ShouldBeNil)
	})

	t.Run("motor GoFor with positive rpm and negative revolutions", func(t *testing.T) {
		// Check with position at 0.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 255, 253, 128, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, 50.0, -3.2, nil), test.ShouldBeNil)

		// Check with position at 4.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 0, 159, 255},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 8, 98, 98, 7}, // Can be gibberish
				{0, 0, 3, 32, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, 50.0, -3.2, nil), test.ShouldBeNil)

		// Check with position at 1.2 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 255, 251, 200, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 240, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, 50.0, -6.6, nil), test.ShouldBeNil)
	})

	t.Run("motor GoFor with negative rpm and negative revolutions", func(t *testing.T) {
		// Check with position at 0.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 2, 128, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, -50.0, -3.2, nil), test.ShouldBeNil)

		// Check with position at 4.0 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 5, 160, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 8, 98, 98, 7}, // Can be gibberish
				{0, 0, 3, 32, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, -50.0, -3.2, nil), test.ShouldBeNil)

		// Check with position at 1.2 revolutions
		//nolint:dupl
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{33, 0, 0, 0, 0},
				{33, 0, 0, 0, 0},
				{160, 0, 0, 0, 0},
				{164, 0, 0, 21, 8},   // a1
				{166, 0, 0, 21, 8},   // aMax
				{170, 0, 0, 21, 8},   // d1
				{168, 0, 0, 21, 8},   // dMax
				{163, 0, 0, 0, 1},    // vStart
				{171, 0, 0, 0, 10},   // vStop
				{165, 0, 2, 17, 149}, // v1
				{167, 0, 0, 211, 213},
				{173, 0, 6, 24, 0},
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 240, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 2, 0},
			},
		)
		test.That(t, motorDep.GoFor(ctx, -50.0, -6.6, nil), test.ShouldBeNil)
	})

	t.Run("motor GoFor with zero rpm", func(t *testing.T) {
		test.That(t, motorDep.GoFor(ctx, 0, 1, nil), test.ShouldBeError, motor.NewZeroRPMError())
	})

	t.Run("motor is on testing", func(t *testing.T) {
		// Off
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 4, 0},
				{0, 0, 0, 4, 0},
			},
		)
		on, powerPct, err := motorDep.IsPowered(ctx, nil)
		test.That(t, on, test.ShouldEqual, false)
		test.That(t, powerPct, test.ShouldEqual, -0.25)
		test.That(t, err, test.ShouldBeNil)

		// On
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 5, 0},
				{0, 0, 0, 0, 0},
			},
		)
		on, powerPct, err = motorDep.IsPowered(ctx, nil)
		test.That(t, on, test.ShouldEqual, true)
		test.That(t, powerPct, test.ShouldEqual, -0.25)
		test.That(t, err, test.ShouldBeNil)
	})

	t.Run("motor zero testing", func(t *testing.T) {
		// No offset (and when actually off)
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
				{160, 0, 0, 0, 3},
				{173, 0, 0, 0, 0},
				{161, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 0, 4, 0},
				{0, 0, 0, 4, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
			},
		)
		test.That(t, motorDep.ResetZeroPosition(ctx, 0, nil), test.ShouldBeNil)

		// No offset (and when actually on)
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
			},
			[][]byte{
				{0, 0, 128, 0, 4},
				{0, 0, 0, 0, 0},
			},
		)
		test.That(t, motorDep.ResetZeroPosition(ctx, 0, nil), test.ShouldNotBeNil)

		// 3.1 offset (and when actually off)
		fakeSpiHandle.AddExpectedRx(
			[][]byte{
				{53, 0, 0, 0, 0},
				{53, 0, 0, 0, 0},
				{160, 0, 0, 0, 3},
				{173, 255, 253, 148, 0},
				{161, 255, 253, 148, 0},
			},
			[][]byte{
				{0, 0, 0, 4, 0},
				{0, 0, 0, 4, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
			},
		)
		test.That(t, motorDep.ResetZeroPosition(ctx, 3.1, nil), test.ShouldBeNil)
	})

	//nolint:dupl
	t.Run("test over-limit current settings", func(*testing.T) {
		mc.HoldDelay = 9999
		mc.RunCurrent = 9999
		mc.HoldCurrent = 9999

		fakeSpiHandle, fakeSpi := newFakeSpi(t)

		// These are the setup register writes
		fakeSpiHandle.AddExpectedTx([][]byte{
			{236, 0, 1, 0, 195},
			{176, 0, 15, 31, 31}, // Last three are delay, run, and hold
			{237, 0, 0, 0, 0},
			{164, 0, 0, 21, 8},
			{166, 0, 0, 21, 8},
			{170, 0, 0, 21, 8},
			{168, 0, 0, 21, 8},
			{163, 0, 0, 0, 1},
			{171, 0, 0, 0, 10},
			{165, 0, 2, 17, 149},
			{177, 0, 0, 105, 234},
			{167, 0, 0, 0, 0},
			{160, 0, 0, 0, 1},
			{161, 0, 0, 0, 0},
		})

		m, err := makeMotor(ctx, deps, mc, name, logger, fakeSpi)
		test.That(t, err, test.ShouldBeNil)
		fakeSpiHandle.ExpectDone()
		test.That(t, m.Close(context.Background()), test.ShouldBeNil)
	})

	t.Run("test under-limit current settings", func(*testing.T) {
		mc.HoldDelay = -9999
		mc.RunCurrent = -9999
		mc.HoldCurrent = -9999

		fakeSpiHandle, fakeSpi := newFakeSpi(t)

		// These are the setup register writes
		fakeSpiHandle.AddExpectedTx([][]byte{
			{236, 0, 1, 0, 195},
			{176, 0, 0, 0, 0}, // Last three are delay, run, and hold
			{237, 0, 0, 0, 0},
			{164, 0, 0, 21, 8},
			{166, 0, 0, 21, 8},
			{170, 0, 0, 21, 8},
			{168, 0, 0, 21, 8},
			{163, 0, 0, 0, 1},
			{171, 0, 0, 0, 10},
			{165, 0, 2, 17, 149},
			{177, 0, 0, 105, 234},
			{167, 0, 0, 0, 0},
			{160, 0, 0, 0, 1},
			{161, 0, 0, 0, 0},
		})

		m, err := makeMotor(ctx, deps, mc, name, logger, fakeSpi)
		test.That(t, err, test.ShouldBeNil)
		fakeSpiHandle.ExpectDone()
		test.That(t, m.Close(context.Background()), test.ShouldBeNil)
	})

	//nolint:dupl
	t.Run("test explicit current settings", func(*testing.T) {
		mc.HoldDelay = 12
		// Currents will be passed as one less, as default is repurposed
		mc.RunCurrent = 27
		mc.HoldCurrent = 14

		fakeSpiHandle, fakeSpi := newFakeSpi(t)

		// These are the setup register writes
		fakeSpiHandle.AddExpectedTx([][]byte{
			{236, 0, 1, 0, 195},
			{176, 0, 12, 26, 13}, // Last three are delay, run, and hold
			{237, 0, 0, 0, 0},
			{164, 0, 0, 21, 8},
			{166, 0, 0, 21, 8},
			{170, 0, 0, 21, 8},
			{168, 0, 0, 21, 8},
			{163, 0, 0, 0, 1},
			{171, 0, 0, 0, 10},
			{165, 0, 2, 17, 149},
			{177, 0, 0, 105, 234},
			{167, 0, 0, 0, 0},
			{160, 0, 0, 0, 1},
			{161, 0, 0, 0, 0},
		})

		m, err := makeMotor(ctx, deps, mc, name, logger, fakeSpi)
		test.That(t, err, test.ShouldBeNil)
		fakeSpiHandle.ExpectDone()
		test.That(t, m.Close(context.Background()), test.ShouldBeNil)
	})
}

func TestRampParametersConfig(t *testing.T) {
	t.Run("default config with no ramp_parameters in JSON", func(t *testing.T) {
		// JSON blob with no ramp_parameters field
		jsonBlob := `{
			"spi_bus": "main",
			"chip_select": "40",
			"index": 1,
			"ticks_per_rotation": 200,
			"max_rpm": 500,
			"max_acceleration_rpm_per_sec": 300
		}`

		var cfg Config
		err := json.Unmarshal([]byte(jsonBlob), &cfg)
		test.That(t, err, test.ShouldBeNil)

		// Validate the config
		_, _, err = cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)

		// Verify that RampParameters is nil (not provided in JSON)
		test.That(t, cfg.RampParameters, test.ShouldBeNil)

		// Verify values from JSON were parsed correctly
		test.That(t, cfg.MaxRPM, test.ShouldEqual, 500)
		test.That(t, cfg.MaxAcceleration, test.ShouldEqual, 300)
		// Note: CalFactor and HomeRPM defaults are set in makeMotor, not Validate
	})

	t.Run("partial ramp_parameters in JSON", func(t *testing.T) {
		// JSON blob with only some ramp_parameters fields
		jsonBlob := `{
			"spi_bus": "main",
			"chip_select": "40",
			"index": 1,
			"ticks_per_rotation": 200,
			"max_rpm": 500,
			"max_acceleration_rpm_per_sec": 300,
			"ramp_parameters": {
				"v_start": 5,
				"a_max": 1000
			}
		}`

		var cfg Config
		err := json.Unmarshal([]byte(jsonBlob), &cfg)
		test.That(t, err, test.ShouldBeNil)

		// Validate the config
		_, _, err = cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)

		// Verify that RampParameters is not nil
		test.That(t, cfg.RampParameters, test.ShouldNotBeNil)

		// Verify that specified fields are set
		test.That(t, cfg.RampParameters.VStart, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.VStart, test.ShouldEqual, 5)
		test.That(t, cfg.RampParameters.AMax, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.AMax, test.ShouldEqual, 1000)

		// Verify that unspecified fields remain nil (will be merged with defaults in makeMotor)
		test.That(t, cfg.RampParameters.VStop, test.ShouldBeNil)
		test.That(t, cfg.RampParameters.V1, test.ShouldBeNil)
		test.That(t, cfg.RampParameters.A1, test.ShouldBeNil)
		test.That(t, cfg.RampParameters.D1, test.ShouldBeNil)
		test.That(t, cfg.RampParameters.VMax, test.ShouldBeNil)
		test.That(t, cfg.RampParameters.DMax, test.ShouldBeNil)
	})

	t.Run("full ramp_parameters in JSON", func(t *testing.T) {
		// JSON blob with all ramp_parameters fields
		jsonBlob := `{
			"spi_bus": "main",
			"chip_select": "40",
			"index": 1,
			"ticks_per_rotation": 200,
			"max_rpm": 500,
			"max_acceleration_rpm_per_sec": 300,
			"ramp_parameters": {
				"v_start": 2,
				"v_stop": 15,
				"v1": 5000,
				"a1": 800,
				"d1": 900,
				"v_max": 10000,
				"a_max": 1200,
				"d_max": 1300
			}
		}`

		var cfg Config
		err := json.Unmarshal([]byte(jsonBlob), &cfg)
		test.That(t, err, test.ShouldBeNil)

		// Validate the config
		_, _, err = cfg.Validate("")
		test.That(t, err, test.ShouldBeNil)

		// Verify that RampParameters is not nil
		test.That(t, cfg.RampParameters, test.ShouldNotBeNil)

		// Verify that all fields are set correctly
		test.That(t, cfg.RampParameters.VStart, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.VStart, test.ShouldEqual, 2)
		test.That(t, cfg.RampParameters.VStop, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.VStop, test.ShouldEqual, 15)
		test.That(t, cfg.RampParameters.V1, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.V1, test.ShouldEqual, 5000)
		test.That(t, cfg.RampParameters.A1, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.A1, test.ShouldEqual, 800)
		test.That(t, cfg.RampParameters.D1, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.D1, test.ShouldEqual, 900)
		test.That(t, cfg.RampParameters.VMax, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.VMax, test.ShouldEqual, 10000)
		test.That(t, cfg.RampParameters.AMax, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.AMax, test.ShouldEqual, 1200)
		test.That(t, cfg.RampParameters.DMax, test.ShouldNotBeNil)
		test.That(t, *cfg.RampParameters.DMax, test.ShouldEqual, 1300)
	})
}
