# [`analog-devices` module](https://github.com/viam-modules/analog-devices)

This [analog-devices module](https://app.viam.com/module/viam/analog-devices) implements a analog-devices [TMC5072 chip](https://www.trinamic.com/support/eval-kits/details/tmc5072-bob/) for a stepper motor using the [`rdk:component:motor` API](https://docs.viam.com/appendix/apis/components/motor/).

Whereas a basic low-level stepper driver supported by the [`gpiostepper` model](/components/motor/gpiostepper/) sends power to a stepper motor based on PWM signals from GPIO pins, the TMC5072 chip uses SPI bus to communicate with the board, does some processing on the chip itself, and provides convenient features including StallGuard2<sup>TM</sup>.

## Configure your tmc5072 motor

> [!NOTE]
> Before configuring your motor, you must [create a machine](https://docs.viam.com/cloud/machines/#add-a-new-machine).

Navigate to the [**CONFIGURE** tab](https://docs.viam.com/configure/) of your [machine](https://docs.viam.com/fleet/machines/) in the [Viam app](https://app.viam.com/).
[Add motor / analog-devices:tmc5072 to your machine](https://docs.viam.com/configure/#components).

On the new component panel, copy and paste the following attribute template into your motor's attributes field:

```json
{
  "spi_bus": "<your-spi-bus-index>",
  "chip_select": "<pin-number>",
  "index": "<your-terminal-index>",
  "pins": {
    "en_low": "<int>",
  },
  "ticks_per_rotation": <int>,
  "max_acceleration_rpm_per_sec": <float>,
  "sg_thresh": <int>,
  "home_rpm": <float>,
  "cal_factor": <float>,
  "run_current": <int>,
  "hold_current": <int>,
  "hold_delay": <int>
}
```

### Attributes

The following attributes are available for `viam:analog-devices:tmc5072` motors:

| Attribute | Type | Required? | Description |
| --------- | ---- | --------- | ----------  |
| `spi_bus` | string | **Required** | The index of the SPI bus over which the TMC chip communicates with the board. |
|`chip_select` | string | **Required** | The chip select number (CSN) that the TMC5072 is wired to. For Raspberry Pis, use `"0"` if the CSN is wired to {{< glossary_tooltip term_id="pin-number" text="pin number" >}} 24 (GPIO 8) on the Pi, or use `"1"` if you wire the CSN to pin 26. The board sets this high or low to let the TMC chip know whether to listen for commands over SPI. |
| `pins` | object | **Required** | A structure that holds the pin number you are using for `"en_low"`, the enable pin for the driver chip. |
| `index` | int | **Required** | The index of the part of the chip the motor is wired to. Either `1` or `2`, depending on whether the motor is wired to the "MOTOR1" terminals or the "MOTOR2" terminals, respectively. |
| `ticks_per_rotation` | int | **Required** | Number of full steps in a rotation. 200 (equivalent to 1.8 degrees per step) is very common. If your data sheet specifies this in terms of degrees per step, divide 360 by that number to get ticks per rotation. |
| `max_acceleration_rpm_per_sec` | float | Optional | Set a limit on maximum acceleration in revolutions per minute per second. |
| `sg_thresh` | int | Optional | Stallguard threshold; sets sensitivity of virtual endstop detection when homing. |
| `home_rpm` | float | Optional | Speed in revolutions per minute that the motor will turn when executing a Home() command (through DoCommand()). |
| `cal_factor` | float | Optional | Calibration factor for velocity and acceleration. Compensates for clock source drift when doing time-based calculations. |
| `run_current` | int | Optional | Set current when motor is turning, from 1-32 as a percentage of rsense voltage. Defaults to 15 if omitted or set to 0. |
| `hold_current` | int | Optional | Set current when motor is holding a position, from 1-32 as a percentage of rsense voltage. Defaults to 8 if omitted or set to 0. |
| `hold_delay` | int | Optional | How long to hold full power at a set position before ramping down to `hold_current`. 0=instant powerdown, 1-15=delay * 2^18 clocks, 6 is the default. |

Refer to your motor and motor driver data sheets for specifics.

## Example configuration

### `viam:analog-devices:tmc5072`
```json
  {
      "name": "<your-analog-devices-tmc5072-motor-name>",
      "model": "viam:analog-devices:tmc5072",
      "type": "motor",
      "namespace": "rdk",
      "attributes": {
        "chip_select": "0",
        "index": 1,
        "pins": {
          "en_low": "17"
        },
        "max_acceleration": 10000,
        "max_rpm": 450,
        "spi_bus": "0",
        "ticks_per_rotation": 200
      }
      "depends_on": []
  }
```

### Next Steps
- To test your motor, expand the **TEST** section of its configuration pane or go to the [**CONTROL** tab](https://docs.viam.com/fleet/control/).
- To write code against your motor, use one of the [available SDKs](https://docs.viam.com/sdks/).
- To view examples using a motor component, explore [these tutorials](https://docs.viam.com/tutorials/).

## Extended API

The `TMC5072` model supports additional methods that are not part of the standard Viam motor API (passed through DoCommand()).

For more information on `do_command`, see the [Python SDK Docs](https://python.viam.dev/autoapi/viam/components/component_base/index.html#viam.components.component_base.ComponentBase.do_command), or the Go SDK docs on [Home()](https://pkg.go.dev/go.viam.com/rdk/components/motor/tmcstepper#Motor.Home) and on [DoCommand()](https://pkg.go.dev/go.viam.com/rdk/components/motor/tmcstepper#Motor.DoCommand).

### Home

Home the motor using [TMC's StallGuard<sup>TM</sup>](https://www.trinamic.com/technology/motor-control-technology/stallguard-and-coolstep/) (a builtin feature of this controller).

**Parameters:**

- `ctx` [(Context)](https://pkg.go.dev/context): A Context carries a deadline, a cancellation signal, and other values across API boundaries.

**Returns:**

- [(error)](https://pkg.go.dev/builtin#error): An error, if one occurred.

**Go Example:**

```go
// Home the motor
resp, err := myMotorComponent.DoCommand(ctx, map[string]interface{}{"command": "home"})
```

### Jog

Move the motor indefinitely at the specified RPM.

**Parameters:**

- `ctx` [(Context)](https://pkg.go.dev/context): A Context carries a deadline, a cancellation signal, and other values across API boundaries.
- `rpm` [(float64)](https://pkg.go.dev/builtin#float64): The revolutions per minute at which the motor will turn indefinitely.

**Returns:**

- [(error)](https://pkg.go.dev/builtin#error): An error, if one occurred.

For more information, see the Go SDK Docs on [`Jog`](https://pkg.go.dev/go.viam.com/rdk/components/motor/tmcstepper#Motor.Jog) and on [`DoCommand`](https://pkg.go.dev/go.viam.com/rdk/components/motor/tmcstepper#Motor.DoCommand).

```go
// Run the motor indefinitely at 70 rpm
resp, err := myMotorComponent.DoCommand(ctx, map[string]interface{}{"command": "jog", "rpm": 70})
```
