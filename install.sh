#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva
set -e

echo "Compilando tae..."
go build -o tae main.go

# $PREFIX é uma variável de ambiente nativa e exclusiva do Termux
if [ -n "$PREFIX" ] && [ -d "$PREFIX/bin" ]; then
    echo "Ambiente Termux detectado."
    DEST="$PREFIX/bin/tae"
    mv tae "$DEST"
    chmod +x "$DEST"
else
    echo "Ambiente Linux padrão detectado."
    DEST="/usr/local/bin/tae"
    echo "Isso requer privilégios de root para gravar em $DEST"
    sudo mv tae "$DEST"
    sudo chmod +x "$DEST"
fi

echo "Sucesso! 'tae' instalado em $DEST."
echo "Você já pode executar o comando 'tae' de qualquer lugar."

echo "Configurando o wrapper para navegação de diretórios (tae cd)..."
mkdir -p "$HOME/.tae"
cp tae_wrapper.sh "$HOME/.tae/tae.sh"

BASHRC="$HOME/.bashrc"
ZSHRC="$HOME/.zshrc"
SOURCE_CMD='[ -s "$HOME/.tae/tae.sh" ] && source "$HOME/.tae/tae.sh" # Integração da CLI Tae'

# Configurar bash
if [ -f "$BASHRC" ]; then
    if ! grep -q "tae.sh" "$BASHRC"; then
        echo "" >> "$BASHRC"
        echo "$SOURCE_CMD" >> "$BASHRC"
        echo "Adicionado ao $BASHRC"
    fi
fi

# Configurar zsh se existir
if [ -f "$ZSHRC" ]; then
    if ! grep -q "tae.sh" "$ZSHRC"; then
        echo "" >> "$ZSHRC"
        echo "$SOURCE_CMD" >> "$ZSHRC"
        echo "Adicionado ao $ZSHRC"
    fi
fi

echo "Para utilizar a navegação de diretórios na sessão atual, reinicie o terminal ou execute: source ~/.tae/tae.sh"
