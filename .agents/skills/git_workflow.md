# Fluxo de Trabalho com Git: Commits Atômicos e Convencionais

Este documento estabelece o padrão para as interações do agente com o Git.

## 1. Commits Atômicos
Cada commit deve representar uma única unidade lógica de alteração.
- Não misture várias funcionalidades ou correções independentes em um mesmo commit.
- Use `git add -p` ou selecione arquivos/linhas específicos para garantir que as unidades sejam preservadas, mesmo que várias alterações tenham sido feitas no diretório de trabalho.
- Se uma tarefa envolver várias etapas (ex.: criar um pacote, adicionar um teste e integrar com a CLI), cada etapa deve ter seu próprio commit.

## 2. Commits Convencionais
Todas as mensagens de commit devem seguir a especificação Conventional Commits.

- `feat:` para novas funcionalidades.
- `fix:` para correção de bugs.
- `docs:` para alterações na documentação.
- `test:` para adição ou correção de testes.
- `refactor:` para mudanças no código que não corrigem bug nem adicionam funcionalidade.
- `chore:` para atualizações de tarefas de build, configurações de gerenciadores de pacotes, etc.

## 3. Fluxo de Trabalho
1. Conclua uma unidade lógica de trabalho.
2. Verifique se funciona/compila.
3. Sempre solicite permissão explícita do usuário antes de fazer um commit.
4. Siga para a próxima unidade.
5. Absolutamente nunca execute `git push`.

## 4. Formato da mensagem:
- Primeira linha: <tipo>: descrição breve em pt-br
- Linhas seguintes: explicação da alteração com detalhes do objetivo