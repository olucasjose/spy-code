# ~/.tae/tae.sh
# Interceptador da CLI Tae para navegação de diretórios

tae() {
    if [ "$1" = "cd" ]; then
        if [ -z "$2" ]; then
            echo "Erro: Nome da tag não fornecido. Uso: tae cd <tag>"
            return 1
        fi
        
        local dest
        if ! dest=$(command tae info "$2" --gitroot 2>/dev/null); then
            echo "Erro: Tag '$2' não encontrada."
            return 1
        fi
        
        if [ -z "$dest" ]; then
            echo "Erro: A tag '$2' não possui um Git Root configurado."
            return 1
        fi
        
        cd "$dest" || return 1
    else
        command tae "$@"
    fi
}
