#!/bin/bash

# Swagger 文档生成脚本

set -e

echo "==================================="
echo "   Swagger 文档生成工具"
echo "==================================="
echo ""

# 检查 swag 是否安装
if ! command -v swag &> /dev/null; then
    echo "❌ swag 命令未找到"
    echo ""
    echo "正在安装 swag..."
    go install github.com/swaggo/swag/cmd/swag@latest
    echo "✅ swag 安装完成"
    echo ""
fi

# 进入项目根目录
cd "$(dirname "$0")/.."

echo "📁 当前目录: $(pwd)"
echo ""

# 清理旧文档
if [ -d "docs" ] && [ -f "docs/docs.go" ]; then
    echo "🗑️  清理旧文档..."
    rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
    echo "✅ 清理完成"
    echo ""
fi

# 生成新文档
echo "📝 生成 Swagger 文档..."
swag init -g cmd/devops-cd/main.go -o docs --parseDependency --parseInternal

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Swagger 文档生成成功！"
    echo ""
    echo "生成的文件："
    echo "  - docs/docs.go"
    echo "  - docs/swagger.json"
    echo "  - docs/swagger.yaml"
    echo ""
    echo "访问地址："
    echo "  http://localhost:8080/swagger/index.html"
    echo ""
    echo "使用说明："
    echo "  1. 启动服务: go run cmd/devops-cd/main.go -config=configs/base.yaml"
    echo "  2. 打开浏览器访问上述地址"
    echo "  3. 查看详细使用说明: docs/SWAGGER_GUIDE.md"
    echo ""
else
    echo ""
    echo "❌ 文档生成失败"
    echo ""
    echo "可能的原因："
    echo "  1. Go 版本不匹配"
    echo "  2. 注释格式错误"
    echo "  3. 依赖包缺失"
    echo ""
    echo "解决方案："
    echo "  go clean -cache -modcache"
    echo "  go mod tidy"
    echo "  go install github.com/swaggo/swag/cmd/swag@latest"
    echo ""
    exit 1
fi

