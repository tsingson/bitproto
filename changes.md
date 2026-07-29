# Compiler-go 对齐 Python Compiler 改动记录（2026-07-30）

## 背景
本次工作目标是将 compiler-go（native 渲染链路）尽量对齐 Python 版本 compiler 的行为与输出，并完成回归验证、三端互通验证和稳定性验证。

## 代码修改

### 1) AST 与语法解析对齐
- 文件：compiler-go/native/ast.go
  - `Message` 新增字段：`Extensible bool`
  - `TypeExpr` 新增字段：`Extensible bool`

- 文件：compiler-go/native/parser.go
  - 解析 `message X' { ... }` 的 extensible 标记，并写入 `Message.Extensible`
  - 解析数组类型 `type[N]'` 的 extensible 标记，并写入 `TypeExpr.Extensible`
  - const 解析结果保留在 AST 中（此前已接入）

### 2) 通用渲染辅助对齐
- 文件：compiler-go/native/render_common.go
  - 新增 `toSnakeCase`，用于与 Python compiler 一致的标识符风格处理
  - 新增 C 端 json formatter 命名函数：
    - `msgJsonFormatterName`
    - `aliasJsonFormatterName`
    - `arrayJsonFormatterNameForAlias`
    - `arrayJsonFormatterNameForField`

### 3) C 头/源渲染对齐
- 文件：compiler-go/native/render_c.go
  - 头文件 include guard 改为 Python compiler 风格：
    - `__BITPROTO__{PROTO_NAME}_H__`
  - 新增 `extern "C"` 包裹（C++ 兼容）
  - const 渲染保留（输出为 `#define`）
  - 增加内部函数声明：processor/json formatter（alias/message/array）
  - `BpArray/BpAlias/BpMessage` 的 descriptor 构建中补齐 json formatter 指针，不再使用 `NULL`
  - message descriptor 的 extensible 参数改为动态值（来自 `Message.Extensible`）
  - 生成 array json formatter 函数
  - `Json<Message>` 从 stub 改为真实 `BpJsonFormatContext` + message json formatter 路径

### 4) Go 渲染对齐
- 文件：compiler-go/native/render_go.go
  - imports 增加 `strconv`
  - 增加 `var formatInt = strconv.FormatInt`
  - enum 增加 `String()` 方法（与 Python compiler 产物行为对齐）
  - message `BpProcessor()` 中 extensible 参数改为动态值（来自 `Message.Extensible`）
  - const 渲染保留（输出为 `const NAME TYPE = VALUE`）

### 5) Python 渲染补齐
- 文件：compiler-go/native/render_py.go
  - 补齐 const 输出（类型映射为 Python 表达）：
    - bool -> `bool` 且值 `True/False`
    - string -> `str`
    - int -> `int`

### 6) 测试用例同步
- 文件：compiler-go/native/float_fixed_test.go
  - 头文件 guard 断言更新为新格式：
    - `#ifndef __BITPROTO__SAMPLE_H__`

### 7) Go runtime 模块补齐
- 新增文件：lib/go/go.mod
  - module: `github.com/tsingson/bitproto/lib/go`
  - 用于测试/临时模块构建时稳定引用 lib/go

## 验证结果

### A. Go 侧单测
- 命令：`go test ./...`
- 结果：通过

### B. Python tests 全量
- 命令：`GOFLAGS=-mod=mod PATH="/tmp/bitproto-venv/bin:$PATH" PYTHONPATH="/Users/qinshen/go/zephyrproject/bitproto/lib/py" /tmp/bitproto-venv/bin/python -m pytest tests -v`
- 结果：`102 passed`

### C. 生成一致性验证脚本
- `scripts/validate_go_compiler_c.sh`：通过
  - 普通模式和优化模式下，Go compiler 与 Python compiler 生成的 C/H 文件一致
  - C runtime 行为一致

- `scripts/validate_go_compiler_go.sh`：通过
  - 普通模式和优化模式下，Go compiler 与 Python compiler 生成的 Go 文件一致
  - Go runtime 行为一致

### D. 三端互通验证
- `scripts/verify_go_compiler_interop.sh`：通过
  - C/Go/Python 三端交叉序列化/反序列化一致

### E. 稳定性与可靠性
- `tests/test_encoding/test_native_interop.py` 连续 3 轮复跑：通过
- `tests/test_encoding/test_endian.py`：通过（20 passed）
- `tests/test_encoding/test_tcpip.py`：通过（6 passed）

## 备注
- 部分测试依赖本地环境变量以允许临时 Go module 自动解析（`GOFLAGS=-mod=mod`）以及 Python runtime 库路径（`PYTHONPATH=.../lib/py`）。
- C 编译阶段有少量 `unused parameter` warning，未影响功能正确性与互通结果。
