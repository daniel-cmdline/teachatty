# Features

Registro das features entregues, da mais nova para a mais antiga. Adicione uma entrada a cada commit.

## 2026-10-12 08:43 — Módulo renomeado para `github.com/daniel-cmdline/teachat`

- O `go.mod` passou de `module httpEndpoint` para `module github.com/daniel-cmdline/teachat`, o mesmo caminho do repositório.
- **Por quê:** o nome do módulo é o começo de todo import interno. Com pacotes próprios chegando (`internal/ui`), os imports passam a ser `github.com/daniel-cmdline/teachat/internal/...`. E o `go install github.com/daniel-cmdline/teachat/...@latest` só funciona quando o nome do módulo bate com o caminho do repositório.
- **Por que agora:** hoje é uma linha. Depois de existirem pacotes, renomear exige mexer em todos os imports.
- Nenhuma mudança de comportamento.

## 2026-10-11 10:28 — Trava por sala

- `Room` foi para `cmd/server/room.go`, com mutex próprio e os métodos `join`, `leave` e `broadcast`.
- O mutex global agora protege só o mapa `rooms`. O broadcast trava apenas a própria sala.
- **Por quê:** com uma trava global única, o broadcast numa sala grande bloqueava todas as outras.
- **Resultado:** com 50 mil clientes numa sala recebendo mensagens, a latência numa sala pequena caiu de ~265 ms para ~0,1 ms. O detector de race condition não encontra problemas.

## 2026-10-11 09:45 — Salas e alias

- O cliente recebe as flags `--server`, `--room` e `--alias` (padrões: `localhost:23456`, `geral`, `anonimo`) em vez de uma URL `ws://` crua. A URL é montada internamente.
- O `--server` permite que uma empresa aponte o cliente para a própria instância do teachat.
- As mensagens vão só para quem está na mesma sala de quem enviou, com o alias na frente (`daniel: oi`).
- Entrar numa sala que não existe cria a sala. A sala é apagada quando o último membro sai.

## 2026-10-11 09:34 — Primeiro commit: chat WebSocket

- Servidor (`cmd/server`) e cliente (`cmd/client`) como binários separados num mesmo módulo.
- WebSocket com `gorilla/websocket`. O servidor repassa cada mensagem para todos os outros clientes conectados.
- O cliente envia uma mensagem a cada Enter e mostra as mensagens recebidas em paralelo.
