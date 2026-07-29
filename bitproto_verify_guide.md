# bitproto 三端互通验证指南

## 1. 本次修改摘要

### 1.1 CI 接入三端同测
- 已在 CI workflow 中新增步骤：
  - 名称：`Verify native c/go/py three-end interop stress`
  - 命令：`python tests/test_encoding/test_native_interop.py`
  - 执行条件：仅在 `matrix.python == '3.11'` 下执行，避免矩阵重复耗时。
- 位置：`.github/workflows/ci.yml`

### 1.2 native-py 渲染顺序修复
- 问题：生成的 Python 文件中消息类先引用 `BYTES_LENGTH_*`，但常量在后面定义，导入即 `NameError`。
- 修复：将 `BYTES_LENGTH_*` 常量提前到消息类定义之前输出。
- 位置：`compiler-go/native/render_py.go`

### 1.3 三端脚本生成入口修正
- 问题：优先使用 `dist/bitproto-go` 时，可能命中旧二进制，导致误报。
- 修复：统一改为在 `compiler-go` 目录执行 `go run .` 生成 native c/go/py 产物。
- 位置：`tests/test_encoding/test_native_interop.py`

### 1.4 C 端安全校验修复
- 问题：C 探针 payload 区域初始值与按位 OR 编码路径冲突，可能污染编码结果。
- 修复：在保持前后 canary 的同时，编码前显式清零 payload 区域。
- 位置：`tests/test_encoding/test_native_interop.py`

### 1.5 native-py 回环断言修正
- 问题：`to_json()` 路径依赖 dataclass，native 生成对象并非 dataclass，可能触发 `TypeError`。
- 修复：改为 `decode` 后再次 `encode`，做字节级一致性断言。
- 位置：`tests/test_encoding/test_native_interop.py`

### 1.6 float/double 固定点支持（新增）
- 语法层支持：`float` 与 `double` 可用于消息字段和数组字段。
- 存储/传输语义：两者统一按 `int32` 固定点编码，比例因子为 `1e8`。
- 换算公式：
  - 写入：`stored = round(value * 100000000)`（并做 int32 饱和）
  - 读取：`value = stored / 100000000`
- 可表示范围（8 位小数）：约 `[-21.47483648, 21.47483647]`。

### 1.7 float/double helper API（新增）
- C 生成代码：
  - 标量字段：`Set<Message><Field>Float/Double` 与 `Get<Message><Field>Float/Double`
  - 数组字段：`Set<Message><Field>Float/DoubleAt` 与 `Get<Message><Field>Float/DoubleAt`
- Go 生成代码：
  - 标量字段：`Set<Field>Float/Double` 与 `Get<Field>Float/Double`
  - 数组字段：`Set<Field>Float/DoubleAt(i, v)` 与 `Get<Field>Float/DoubleAt(i)`
- Python 生成代码：
  - 标量字段：`set_field_float/double` 与 `get_field_float/double`
  - 数组字段：`set_field_float/double_at` 与 `get_field_float/double_at`
- 通用转换函数（生成代码中提供）：
  - `BpFloatToInt32` / `BpInt32ToFloat`
  - `bp_float_to_int32` / `bp_int32_to_float`

## 2. 验证步骤（本地与 CI 一致）

### 2.1 编译器 Go 侧单测
```bash
cd compiler-go
go test ./...
```

### 2.2 三端同测（边界 + 随机压力 + C canary）
```bash
cd ..
python tests/test_encoding/test_native_interop.py
```

### 2.3 稳定性复跑（建议至少 3 轮）
```bash
for i in 1 2 3; do
  echo "round=$i"
  python tests/test_encoding/test_native_interop.py
done
```

## 3. C 端开发注意点

1. 编码缓冲区初始化
- 当前 C runtime 的写位逻辑是基于 OR 写入；编码前必须保证 payload 为 0。
- 若做越界保护（canary），应仅填充保护区，payload 独立清零。

2. 边界与符号位
- 对 signed 位宽（如 int24/int32）必须覆盖最小值、最大值、0、-1 的用例。
- 解码后要验证符号扩展是否正确（尤其非 8 整数字段）。

3. 内存安全
- 建议所有 probe 程序都保留前后 canary 校验，至少覆盖 encode/decode 两次边界检查。
- 编译建议保留 `-Wall -Wextra`，并在 CI 中持续运行。

4. 结构体字段赋值
- 枚举、bool、alias 字段赋值应显式转换目标类型，避免隐式整型提升造成歧义。

## 4. Go 端开发注意点

1. 生成入口一致性
- 验证脚本应优先走源码构建路径（`go run .`），避免依赖陈旧 dist 二进制造成“伪回归”。

2. module 隔离
- 原生生成代码测试建议放到临时 module（含 `replace` 到本地 `lib/go`），确保 runtime 与当前代码同步。

3. 编解码回环
- 对每个向量至少验证 `Encode -> Decode` 后对象一致，或 `Decode -> Encode` 字节一致。
- 对数组、嵌套消息、alias/enum 字段要重点覆盖。

4. 有符号字段处理
- 关注 `BpProcessInt` 的符号扩展路径，特别是非 byte 边界位宽和别名嵌套字段。

## 5. 常见故障排查

1. Python 导入时报 `BYTES_LENGTH_*` 未定义
- 检查 native-py 生成文件中常量定义位置是否在消息类之前。

2. native-py 出现数组越界 `IndexError`
- 检查 alias array 默认值是否按固定长度初始化。

3. C 端回环不一致但 canary 正常
- 优先检查 payload 是否在编码前清零。

4. CI 与本地结果不一致
- 检查是否使用了旧 `dist/bitproto-go`，建议统一走 `compiler-go/go run .` 生成。

## 6. 建议的提交前最小检查

```bash
cd compiler-go && go test ./...
cd .. && python tests/test_encoding/test_native_interop.py
```

## 7. float/double 专项验证

```bash
# 1) 主编译器解析（float/double -> int32 语义）
PYTHONPATH=compiler compiler/.venv/bin/python - <<'PY'
from bitproto.parser import parse
from bitproto._ast import Message, MessageField, Int, Array
from bitproto.utils import cast_or_raise

p = parse('tests/test_compiler/parser-cases/float_double.bitproto')
m = cast_or_raise(Message, p.get_member('Reading'))
f1 = cast_or_raise(MessageField, m.get_member('temperature'))
f2 = cast_or_raise(MessageField, m.get_member('humidity'))
f3 = cast_or_raise(MessageField, m.get_member('history'))
assert isinstance(f1.type, Int) and f1.type.cap == 32
assert isinstance(f2.type, Int) and f2.type.cap == 32
a = cast_or_raise(Array, f3.type)
assert isinstance(a.element_type, Int) and a.element_type.cap == 32
print('float/double parser mapping ok')
PY

# 2) native 渲染与 helper API 生成
cd compiler-go && go test ./native -run TestRenderFixedFloatTypes -v
```
