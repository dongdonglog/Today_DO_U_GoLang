.PHONY: docs check-docs test run clean help

SWAG ?= go run github.com/swaggo/swag/cmd/swag

# 生成 Swagger 文档
docs:
	$(SWAG) init -g main.go -o docs --parseDependency --parseInternal
	@echo "Swagger docs generated"

# 检查 Swagger 文档是否包含接口路径
check-docs: docs
	@test -f docs/swagger.json
	@grep -q '"/api/v1/users"' docs/swagger.json
	@grep -q '"/api/v1/login"' docs/swagger.json
	@echo "Swagger docs checked"

# 运行测试
test:
	go test ./...

# 运行服务
run: docs
	go run main.go

# 清理生成的文档
clean:
	rm -rf docs/docs.go docs/swagger.json docs/swagger.yaml

# 帮助
help:
	@echo "Available targets:"
	@echo "  docs       - Generate Swagger documentation"
	@echo "  check-docs - Generate docs and verify important paths"
	@echo "  test       - Run tests"
	@echo "  run        - Run the server with docs"
	@echo "  clean      - Remove generated docs"
	@echo "  help       - Show this help message"
