# [`analog-devices` module](https://github.com/viam-modules/analog-devices)

This [analog-devices module](https://app.viam.com/module/viam/analog-devices) implements an analog-devices [TMC5072 chip](https://www.trinamic.com/support/eval-kits/details/tmc5072-bob/) for a stepper motor using the [`rdk:component:motor` API](https://docs.viam.com/appendix/apis/components/motor/), and an analog-devices [ADXL345 digital accelerometer](https://www.analog.com/en/products/adxl345.html) using the [`rdk:component:movement_sensor` API](https://docs.viam.com/appendix/apis/components/movement-sensor/).

The TMC5072 chip uses SPI bus to communicate with the board, does some processing on the chip itself, and provides convenient features including StallGuard2<sup>TM</sup>.

See [Configure your tmc5072 motor](#Configure-your-tmc5072-motor) or [Configure your adxl345 movement sensor](#Configure-your-adxl345-movement-sensor) for more information on configuring these components with Viam.

> [!NOTE]
> Before configuring your motor or movement sensor, you must [create a machine](https://docs.viam.com/cloud/machines/#add-a-new-machine).

Navigate to the [**CONFIGURE** tab](https://docs.viam.com/configure/) of your [machine](https://docs.viam.com/fleet/machines/) in the [Viam app](https://app.viam.com/).
[Add motor / analog-devices:tmc5072 to your machine](https://docs.viam.com/configure/#components).
[Add movement_sensor / analog-devices:adxl345 to your machine](https://docs.viam.com/configure/#components).

## Configure your tmc5072 motor

On the new component panel, copy and fill in the following required attributes in:
```json
{
  "spi_bus": "0",
  "chip_select" : "0",
  "index" : "1",
  "ticks_per_rotation" : 100
}
```

### Attributes

The following attributes are available for `viam:analog-devices:tmc5072` motors:

| Attribute | Type | Required? | Description |
| --------- | ---- | --------- | ----------  |
| `spi_bus` | string | **Required** | The index of the SPI bus over which the TMC chip communicates with the board. |
|`chip_select` | string | **Required** | The pin on the board that allows for spi chip selects, that the TMC5072 is wired to. For example, on a Raspberry Pi, use `"0"` if the CSN is wired to the physical pin number 24 on the Pi, or use `"1"` if you wire the Chip Select to pin 26. The board sets this high or low to let the TMC chip know whether to listen for commands over SPI. |
| `index` | int | **Required** | The index of the part of the chip the motor is wired to. Either `1` or `2`, depending on whether the motor is wired to the "MOTOR1" terminals or the "MOTOR2" terminals, respectively. |
| `ticks_per_rotation` | int | **Required** | Number of full steps in a rotation. 200 (equivalent to 1.8 degrees per step) is very common. If your data sheet specifies this in terms of degrees per step, divide 360 by that number to get ticks per rotation. |
| `pins` | object | **Optional** | A structure that holds the pin number you are using for `"en_low"`, the enable pin for the driver chip. |
| `max_acceleration_rpm_per_sec` | float | Optional | Set a limit on maximum acceleration in revolutions per minute per second. |
| `sg_thresh` | int | Optional | Stallguard threshold; sets sensitivity of virtual endstop detection when homing. |
| `home_rpm` | float | Optional | Speed in revolutions per minute that the motor will turn when executing a Home() command (through DoCommand()). |
| `cal_factor` | float | Optional | Calibration factor for velocity and acceleration. Compensates for clock source drift when doing time-based calculations. |
| `run_current` | int | Optional | Set current when motor is turning, from 1-32 as a percentage of rsense voltage. Defaults to 15 if omitted or set to 0. |
| `hold_current` | int | Optional | Set current when motor is holding a position, from 1-32 as a percentage of rsense voltage. Defaults to 8 if omitted or set to 0. |
| `hold_delay` | int | Optional | How long to hold full power at a set position before ramping down to `hold_current`. 0=instant powerdown, 1-15=delay * 2^18 clocks, 6 is the default. |

Refer to your motor and motor driver data sheets for specifics.

### Full Config with all optional Attributes
```json
{
  "spi_bus": "<your-spi-bus-index>",
  "chip_select": "<pin-number>",
  "index": <your-terminal-index>,
  "pins": {
    "en_low": "<int>"
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

## Configure your adxl345 movement sensor

This three axis accelerometer supplies linear acceleration data, supporting the `LinearAcceleration` method.

On the new component panel, copy and paste the following attribute template into your movement sensor's attributes field:

```json
  "i2c_bus" : "0"
```

You can configure additional attributes for the sensor, shown in the example json and explained in the attributes table below.  
### Attributes

The following attributes are available for `viam:analog-devices:adxl345` movement sensors:

| Attribute | Type | Required? | Description |
| --------- | ---- | --------- | ----------  |
| `i2c_bus` | string | **Required** | The index of the I2C bus on the board your device is connected to. Often a number. <br> Example: "2"  |
| `use_alternate_i2c_address` | bool | Optional | Depends on whether you wire SDO low (leaving the default address of 0x53) or high (making the address 0x1D). If high, set true. If low, set false or omit the attribute. <br> Default: `false` |
| `board` | string | Optional | The `name` of the [board](https://docs.viam.com/components/board/) to which the device is wired. Only needed if you've configured any interrupt functionality. |
| `tap` | object | Optional | Holds the configuration values necessary to use the tap detection interrupt on the ADXL345. See [Tap attributes](#tap-attributes). |
| `free_fall` | object | Optional | Holds the configuration values necessary to use the free-fall detection interrupt on the ADXL345. See [Freefall attributes](#freefall-attributes). |

### Example Full Config
```json
{
    "i2c_bus": "<your-i2c-bus-index-on-board>",
    "board": "<your-board-name>",
    "use_alternate_i2c_address": <boolean>,
    "tap": {
      "accelerometer_pin": <int>,
      "interrupt_pin": "<your-digital-interrupt-name-on-board>",
      "exclude_x": <boolean>,
      "exclude_y": <boolean>,
      "exclude_z": <boolean>,
      "threshold": <float>,
      "dur_us": <float>
    },
    "free_fall": {
      "accelerometer_pin": <int>,
      "interrupt_pin": "<your-digital-interrupt-name-on-board>",
      "threshold": <float>,
      "time_ms": <float>
    }
}
```

### Tap attributes

Inside the `tap` object, you can include the following attributes:

| Name                | Type   | Required? | Description |
| ------------------- | ------ | --------- | ----------- |
| `accelerometer_pin` | int    | **Required** | On the accelerometer you can choose to send the interrupts to int1 or int2. Specify this by setting this config value to `1` or `2`. |
| `interrupt_pin`     | string | **Required** | The `name` of the digital interrupt you configured for the pin on the [board](https://docs.viam.com/components/board/) wired to the `accelerometer_pin`. |
| `exclude_x`         | bool   | Optional     | Tap detection defaults to all three axes. Exclude the x axis by setting this to true. <br> Default: `false` |
| `exclude_y`         | bool   | Optional     | Tap detection defaults to all three axes. Exclude the y axis by setting this to true. <br> Default: `false` |
| `exclude_z`         | bool   | Optional     | Tap detection defaults to all three axes. Exclude the z axis by setting this to true. <br> Default: `false` |
| `threshold`         | float  | Optional     | The magnitude of the threshold value for tap interrupt (in milligrams, between `0` and `15,937`). <br> Default: `3000` |
| `dur_us`            | float  | Optional     | Unsigned time value representing maximum time that an event must be above the `threshold` to qualify as a tap event (in microseconds, between 0 and 159,375). <br> Default: `10000` |

### Freefall attributes

Inside the `freefall` object, you can include the following attributes:

| Name                | Type   | Required? | Description |
| ------------------- | ------ | --------- | ----------- |
| `accelerometer_pin` | int    | **Required** | On the accelerometer you can choose to send the interrupts to int1 or int2. Specify this by setting this config value to `1` or `2`. |
| `interrupt_pin`     | string | **Required** | The `name` of the digital interrupt you configured for the pin on the [board](https://docs.viam.com/components/board/) wired to the `accelerometer_pin`. |
| `threshold`         | float  | Optional     | The acceleration on each axis is compared with this value to determine if a free-fall event occurred (in milligrams, between `0` and `15,937`). <br> Default: `437.5` |
| `time_ms`           | float  | Optional     | Unsigned time value representing the minimum time that the value of all axes must be less than `threshold` to generate a free-fall interrupt (in milliseconds, between 0 and 1,275). <br> Default: `160` |

## Example configuration

### `viam:analog-devices:adxl345`
```json
  {
      "name": "<your-analog-devices-adxl345-movement-sensor-name>",
      "model": "viam:analog-devices:adxl345",
      "type": "movement_sensor",
      "namespace": "rdk",
      "attributes": {
        "i2c_bus": "2",
        "use_alternate_i2c_address": false
      }
      "depends_on": []
  }
```

### Next Steps
- To test your movement sensor, expand the **TEST** section of its configuration pane or go to the [**CONTROL** tab](https://docs.viam.com/fleet/control/).
- To write code against your movement sensor, use one of the [available SDKs](https://docs.viam.com/sdks/).
- To view examples using a movement sensor component, explore [these tutorials](https://docs.viam.com/tutorials/).
