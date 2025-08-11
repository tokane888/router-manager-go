#!/bin/bash

# mcp install

# context7が存在しない場合のみ追加
if ! claude mcp list 2>/dev/null | grep -q "context7"; then
    echo "Adding MCP server context7..."
    claude mcp add --transport http context7 https://mcp.context7.com/mcp
else
    echo "MCP server context7 already exists, skipping..."
fi

# cipherが存在しない場合のみ追加
if ! claude mcp list 2>/dev/null | grep -q "cipher"; then
    # APIキーの存在確認
    if [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$OLLAMA_BASE_URL" ]; then
        echo "⚠️  WARNING: No API keys found for cipher MCP server."
        echo "   cipher requires one of: ANTHROPIC_API_KEY or OLLAMA_BASE_URL"
        echo "   Set the API key in your host environment (~/.zshrc or ~/.bashrc) and rebuild the DevContainer."
        echo "   Skipping cipher MCP server installation..."
    else
        echo "Adding MCP server cipher..."
        claude mcp add --transport stdio cipher -- cipher --mode mcp
    fi
else
    echo "MCP server cipher already exists, skipping..."
fi

# serenaが存在しない場合のみ追加
if ! claude mcp list 2>/dev/null | grep -q "serena"; then
    echo "Adding MCP server serena..."
    claude mcp add serena -- uvx --from git+https://github.com/oraios/serena serena start-mcp-server --context ide-assistant --project $(pwd)
else
    echo "MCP server serena already exists, skipping..."
fi
