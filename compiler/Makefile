default: lint

install-dev-requirements:
	uv pip install -r requirements_dev.txt

lint:
	uv run mypy bitproto
	uv run isort -c bitproto
	uv run black --check bitproto

reformat:
	uv run black bitproto
	uv run isort bitproto

clean-pip-package:
	rm -rf dist build *egg-info

build-pip-package: clean-pip-package
	uv build

upload-pip-package: build-pip-package
	uv publish

build-go-compiler:
	cd ../compiler-go && go build -o ../dist/bitproto-go .

verify-go-compiler-c:
	bash ../scripts/validate_go_compiler_c.sh

verify-go-compiler-go:
	bash ../scripts/validate_go_compiler_go.sh

verify-go-compiler-interop:
	bash ../scripts/verify_go_compiler_interop.sh
