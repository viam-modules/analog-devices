# [`analog-devices` module](https://github.com/viam-modules/analog-devices)

This [analog-devices module](https://app.viam.com/module/viam/analog-devices) implements a analog-devices [tmc5072 motor](<LINK TO HARDWARE>), used for <DESCRIPTION> using the [`rdk:component:motor` API](https://docs.viam.com/appendix/apis/components/motor/).

> [!NOTE]
> Before configuring your motor, you must [create a machine](https://docs.viam.com/cloud/machines/#add-a-new-machine).

## Configure your tmc5072 motor

Navigate to the [**CONFIGURE** tab](https://docs.viam.com/configure/) of your [machine](https://docs.viam.com/fleet/machines/) in the [Viam app](https://app.viam.com/).
[Add motor / analog-devices:tmc5072 to your machine](https://docs.viam.com/configure/#components).

On the new component panel, copy and paste the following attribute template into your motor's attributes field:

```json
{
  <ATTRIBUTES>
}
```

### Attributes

The following attributes are available for `viam:analog-devices:tmc5072` motors:

<EXAMPLE !!>
| Attribute | Type | Required? | Description |
| --------- | ---- | --------- | ----------  |
| `i2c_bus` | string | **Required** | The index of the I<sup>2</sup>C bus on the board that the motor is wired to. |
| `i2c_address` | string | Optional | Default: `0x77`. The [I<sup>2</sup>C device address](https://learn.adafruit.com/i2c-addresses/overview) of the motor. |

## Example configuration

### `viam:analog-devices:tmc5072`
```json
  {
      "name": "<your-analog-devices-tmc5072-motor-name>",
      "model": "viam:analog-devices:tmc5072",
      "type": "motor",
      "namespace": "rdk",
      "attributes": {
      },
      "depends_on": []
  }
```

### Next Steps
- To test your motor, expand the **TEST** section of its configuration pane or go to the [**CONTROL** tab](https://docs.viam.com/fleet/control/).
- To write code against your motor, use one of the [available SDKs](https://docs.viam.com/sdks/).
- To view examples using a motor component, explore [these tutorials](https://docs.viam.com/tutorials/).