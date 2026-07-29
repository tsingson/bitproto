# bitproto 语法与限制指南

## 1. 目标与范围

本文档用于说明：
- bitproto 的核心语法
- 类型系统与编码规则
- 常见约束与限制
- float/double（你提到的 fleat/double）在当前实现中的固定点语义与精度影响

该指南以仓库当前实现为准，适用于 C / Go / Python 代码生成与互通验证。

## 2. 最小可用语法

一个最小 schema：

```bitproto
proto demo

message Data {
    uint8 id = 1
    bool ok = 2
}
```

要点：
- 文件必须有 proto 名称。
- message 字段需要字段号（= 后面的数字）。
- 同一 message 内字段号必须唯一。

## 3. 顶层声明

bitproto 顶层常见声明：
- proto
- import
- option
- const
- type
- enum
- message

示例：

```bitproto
proto telem

const N = 4

type Timestamp = int64

enum Status : uint3 {
    STATUS_UNKNOWN = 0
    STATUS_OK = 1
    STATUS_WARN = 2
}

message Sample {
    Timestamp ts = 1
    Status status = 2
    int16[N] values = 3
}
```

## 4. 类型系统

### 4.1 基础类型

当前支持的基础类型（语法层）：
- bool
- byte
- uintN（N 为位宽）
- intN（N 为位宽）
- float
- double

位宽规则：
- uintN / intN 的 N 取值范围为 1 到 64。

### 4.2 枚举

- 枚举底层类型使用 uintN（无符号位宽）。
- 枚举值通常使用非负整数。

示例：

```bitproto
enum Mode : uint2 {
    MODE_A = 0
    MODE_B = 1
    MODE_C = 2
}
```

### 4.3 数组

- 仅支持定长数组。
- 数组容量必须是正整数（可由常量计算得到）。

示例：

```bitproto
const WINDOW = 8

message FFT {
    int16[WINDOW] re = 1
    int16[WINDOW] im = 2
}
```

### 4.4 别名

- 可为基础类型、数组、消息等命名。

示例：

```bitproto
type Timestamp = int64
type Vec3 = int32[3]
```

## 5. 编码模型

bitproto 是位级定长编码：
- 每个字段占用位宽在 schema 中可静态确定。
- 不携带可变长度头，不使用动态内存来描述字段尺寸。
- 适合嵌入式、带宽敏感场景。

## 6. float/double 当前实现语义（重点）

## 6.1 不是 IEEE754 直接透传

当前实现中，float 和 double 都不是按 IEEE754 bit pattern 直接上传输。

当前语义为固定点：
- 线上的真实存储类型：有符号 int32
- 缩放因子：1e8
- 规则：
  - 写入：stored = round(value * 100000000)
  - 读取：value = stored / 100000000

## 6.2 精度与范围限制

因为落到 int32，范围受限：
- 最小值约 -21.47483648
- 最大值约 21.47483647

并且存在精度影响：
- 小数保留 8 位，超过部分会四舍五入。
- 超范围时会饱和到 int32 最值。
- NaN / Inf 不作为特殊编码值传输，业务应在写入前自行规避。

这就是你提到的 float/double 精度降低，本质是为了跨语言确定性与 wire 兼容性。

## 6.3 代码生成辅助 API

为降低业务层心智负担，native 生成代码提供 helper API。

C：
- 标量：SetMessageFieldFloat/Double, GetMessageFieldFloat/Double
- 数组：SetMessageFieldFloat/DoubleAt, GetMessageFieldFloat/DoubleAt

Go：
- 标量：SetFieldFloat/Double, GetFieldFloat/Double
- 数组：SetFieldFloat/DoubleAt, GetFieldFloat/DoubleAt

Python：
- 标量：set_field_float/double, get_field_float/double
- 数组：set_field_float/double_at, get_field_float/double_at

同时提供转换函数：
- BpFloatToInt32 / BpInt32ToFloat
- bp_float_to_int32 / bp_int32_to_float

## 7. 常见限制与注意事项

1. 全类型定长
- 不支持可变长字符串、可变长数组、map 等可变结构。

2. 兼容性需要 schema 管理
- 位布局固定，字段调整需严格管理版本演进策略。

补充说明：
- 字段编号、数组容量、消息大小等约束由编译器在解析/校验阶段执行，
    请以当前仓库测试用例与校验错误提示为准。

3. TCP 传输需自行分帧
- TCP 是字节流，不是消息边界。
- 请使用长度头与 read_exact/write_exact 逻辑，避免粘包拆包问题。

4. 跨语言互通要做回归
- 建议每次改 schema 或生成器后运行三端同测与 TCP/IP 用例。

## 8. 推荐验证流程

1. 编译器与生成器单测
- 在 compiler-go 目录执行 go test ./...

2. 三端 native 同测
- 在仓库根目录执行 python tests/test_encoding/test_native_interop.py

3. TCP/IP C<->Go 防粘包回归
- 执行 tests/test_encoding/test_tcpip.py 对应场景（包含 fragmented/concurrent/异常帧）

## 9. 何时不建议使用 float/double

如果你的场景满足以下任一条件，不建议使用当前 fixed-point float/double：
- 需要超过约 21.47 的数值范围
- 需要保留超过 8 位小数
- 需要保留 IEEE754 特性（NaN、Inf、不同舍入语义）

这类场景建议：
- 直接在 schema 中使用 int64/int32 表示业务缩放值，并在业务层自定义比例与溢出策略。

## 10. 相关文档

- 三端互通与回归指南：bitproto_verify_guide.md
- C<->Go TCP/IP 防粘包指南：bitproto-c-to-go-tcpip-guide.md
- Go 编译器验证记录：go-compiler.md

## 11. 端到端实战（schema 到三语言）

本节给出一条可直接执行的最短路径，覆盖：
- 编写 schema
- 生成 C / Go / Python 代码
- 使用 helper API 写入 float/double
- 编码并做基础互通检查

### 11.1 示例 schema

```bitproto
proto telem

message Sensor {
    float temperature = 1
    double humidity = 2
    float[3] history = 3
}
```

### 11.2 生成代码

建议优先使用仓库内 go-compiler（避免环境差异）：

```bash
mkdir -p /tmp/telem-out
cd compiler-go
go run . native-c ../example/example.bitproto /tmp/telem-out
go run . native-go ../example/example.bitproto /tmp/telem-out
go run . native-py ../example/example.bitproto /tmp/telem-out
```

如果你要针对上面的 `telem` schema，请把输入文件路径改为自己的 `.bitproto` 文件。

### 11.3 C 侧写入与读取（示意）

```c
struct Sensor s = {0};

SetSensorTemperatureFloat(&s, 23.45678901);
SetSensorHumidityDouble(&s, 56.12345678);
SetSensorHistoryFloatAt(&s, 0, 23.1);
SetSensorHistoryFloatAt(&s, 1, 23.2);
SetSensorHistoryFloatAt(&s, 2, 23.3);

double t = GetSensorTemperatureFloat(&s);
double h = GetSensorHumidityDouble(&s);

unsigned char buf[BYTES_LENGTH_SENSOR] = {0};
EncodeSensor(&s, buf);
```

### 11.4 Go 侧写入与读取（示意）

```go
s := &Sensor{}

s.SetTemperatureFloat(23.45678901)
s.SetHumidityDouble(56.12345678)
_ = s.SetHistoryFloatAt(0, 23.1)
_ = s.SetHistoryFloatAt(1, 23.2)
_ = s.SetHistoryFloatAt(2, 23.3)

t := s.GetTemperatureFloat()
h := s.GetHumidityDouble()

payload := s.Encode()
_ = t
_ = h
_ = payload
```

### 11.5 Python 侧写入与读取（示意）

```python
s = Sensor()

s.set_temperature_float(23.45678901)
s.set_humidity_double(56.12345678)
s.set_history_float_at(0, 23.1)
s.set_history_float_at(1, 23.2)
s.set_history_float_at(2, 23.3)

t = s.get_temperature_float()
h = s.get_humidity_double()

payload = s.encode()
```

### 11.6 最小互通检查建议

1. 固定一组输入（包含正数、负数、边界值、超范围值）。
2. 在 C/Go/Python 分别编码，比较字节流是否一致。
3. 互相解码后再编码，检查字节流回环一致。
4. 对 float/double 字段检查误差不超过 `0.5 / 1e8`（即一次四舍五入误差）。

### 11.7 常见误区

1. 把 float/double 当成 IEEE754 透传。
- 当前实现不是该语义，而是 fixed-point int32。

2. 忽略范围上限。
- 约 `[-21.47483648, 21.47483647]` 之外会饱和。

3. 直接比较浮点文本字符串。
- 建议比较字节流或使用误差阈值比较。
