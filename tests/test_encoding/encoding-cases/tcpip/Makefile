NAME=tcpip

BP_FILENAME=drone.bitproto
BP_C_FILENAME=drone_bp.c
BP_GO_FILENAME=drone_bp.go
FRAME_BP_FILENAME=frame_header.bitproto
FRAME_C_FILENAME=frame_header_bp.c
FRAME_GO_FILENAME=frame_header_bp.go
BP_LIB_DIR=../../../../../lib/c
BP_LIC_C_PATH=$(BP_LIB_DIR)/bitproto.c
BPTOOL=PYTHONPATH=../../../../compiler ../../../../compiler/.venv/bin/python -m bitproto._main

C_SOURCE_FILE=main.c
C_SOURCE_FILE_LIST=$(C_SOURCE_FILE) $(BP_C_FILENAME) frame/$(FRAME_C_FILENAME) $(BP_LIC_C_PATH)

MODE_SUFFIX?=
C_BIN=client$(MODE_SUFFIX)
GO_BIN=server$(MODE_SUFFIX)

CC_OPTIMIZATION_ARG?=
OPTIMIZATION_MODE_ARGS?=

bp-c:
	@mkdir -p c/frame
	@$(BPTOOL) c $(BP_FILENAME) c/ $(OPTIMIZATION_MODE_ARGS)
	@$(BPTOOL) c $(FRAME_BP_FILENAME) c/frame/ $(OPTIMIZATION_MODE_ARGS)

bp-go:
	@mkdir -p go/framebp
	@$(BPTOOL) go $(BP_FILENAME) go/bp/ $(OPTIMIZATION_MODE_ARGS)
	@$(BPTOOL) go $(FRAME_BP_FILENAME) go/framebp/ $(OPTIMIZATION_MODE_ARGS)

build-c: bp-c
	@cd c && $(CC) $(C_SOURCE_FILE_LIST) -I. -Iframe -I$(BP_LIB_DIR) -o $(C_BIN) $(CC_OPTIMIZATION_ARG)

build-go: bp-go
	@cd go && go build -o $(GO_BIN)

clean:
	@rm -fr c/$(C_BIN) go/$(GO_BIN) c/*_bp.* go/bp/*_bp.*
