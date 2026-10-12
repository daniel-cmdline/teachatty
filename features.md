# Features

Registro das features entregues, da mais nova para a mais antiga. Adicione uma entrada a cada commit.

## 2026-10-12 09:50 — Cores por alias

- Na área de mensagens, o alias de cada pessoa aparece numa cor própria, escolhida pelo hash do alias (FNV-32a) numa paleta de 12 cores. O mesmo alias tem sempre a mesma cor, em qualquer cliente, sem mudar o servidor nem o protocolo.
- Suas próprias mensagens: alias em negrito, na cor normal do texto. Vale só para as linhas que você enviou, não para o alias: se outra pessoa entra com o seu alias, ela aparece colorida, não em negrito.
- Só o alias é colorido; o texto da mensagem continua na cor normal.
- **Segurança:** para colorir, o tview passou a interpretar tags como `[red]`. Tudo que vem de fora (alias, texto, nome da sala no título) passa por `tview.Escape`, então quem manda `[red]oi` continua vendo e mostrando `[red]oi` literal.
- **Limites:** com 12 cores, colisões acontecem (5 pessoas numa sala: ~60% de chance de duas terem a mesma cor). O cliente acha o alias cortando a linha no primeiro `": "`, porque o servidor manda `alias: texto` como string única; um alias com `": "` dentro é cortado errado. Os dois somem quando as mensagens virarem estruturadas (JSON) no protocolo.
- Testado ao vivo com 5 clientes na mesma sala: cores diferentes por alias, negrito só nas próprias mensagens, `[red]oi` aparecendo literal.
- **Docs (HTML): pendente.** Atualizar o site e a documentação junto com as próximas features de UI.

## 2026-10-12 09:40 — Site e documentação separados

- A página única `teachat-por-dentro.html` virou duas, em `docs/`, com o CSS compartilhado em `docs/style.css`:
  - `index.html`, o site: o que é o teachat, a tela, os princípios, os números medidos, a animação de como uma mensagem viaja, como começar e o que vem aí. Botão **docs** no topo e no hero.
  - `docs.html`, a documentação, com barra lateral: visão geral, começando, referência do cliente (flags, teclas, códigos de saída) e do servidor (endpoint, formato das mensagens, ciclo de vida das salas), arquitetura, protocolo, servidor e cliente por dentro, capacidade, limitações e roadmap.
- **Por quê:** a página única misturava apresentação com referência. Quem quer conhecer o projeto e quem quer consultar como algo funciona procuram coisas diferentes.
- Os docs agora estão no git. O `.gitignore`, que só ignorava a página antiga, foi apagado.
- Limitações novas documentadas: o servidor aceita WebSocket de qualquer origem e não limita o tamanho das mensagens.

## 2026-10-12 09:26 — Tela cheia no cliente (tview)

- O cliente virou uma tela cheia feita com `tview`, no pacote novo `internal/ui`: mensagens em cima, numa área com o título `sala · alias`, e uma caixa de digitação embaixo.
- **O atropelo acabou:** a mensagem recebida não escreve mais por cima do que você está digitando. O terminal fica em modo raw e só a goroutine da tela desenha. A goroutine de recebimento entrega cada mensagem com `QueueUpdateDraw`.
- Enter envia. Sua própria mensagem aparece como `alias: texto`, igual às dos outros. O servidor continua sem devolver a mensagem para quem enviou; o cliente mostra a sua localmente.
- A área de mensagens segue as mais novas. PgUp e PgDn rolam. Com PgUp ela para de seguir, para dar para ler o passado sem ser puxado de volta. Digitar qualquer coisa volta para o fim.
- A caixa cresce com o texto, de 1 até 5 linhas, e volta para 1 depois do Enter. Ela quebra a linha em qualquer caractere, não só entre palavras, porque assim a altura sai de uma conta: largura do texto na tela ÷ largura da caixa.
- Ctrl+C sai (antes era Ctrl+D). Se a conexão cai, a tela fecha, o terminal volta ao normal e só então aparece `conexão encerrada`. O `log.Fatal` de antes sairia sem restaurar o terminal.
- O `main.go` do cliente só conhece `ui.New`, `Print`, `Run` e `Stop`. Tudo que é tview fica em `internal/ui`.
- **Bug achado no teste ao vivo:** o `TextView` do tview não acompanha texto novo por padrão. Quando a tela enchia, as mensagens novas ficavam escondidas até apertar PgDn. Resolvido com `ScrollToEnd()`.
- **Dependências novas:** `rivo/tview` v0.42.0, `gdamore/tcell/v2` v2.8.1, `rivo/uniseg` (largura do texto na tela) e as indiretas delas. Tudo Go puro: o cliente continua um binário único.
- **Quebrou:** `echo oi | teachat` falha, porque o tview exige um terminal. Volta com o fallback linha a linha, primeiro passo do modo pipe.
- Sem mouse, de propósito: com o mouse ligado, selecionar e copiar texto do chat passaria a exigir Shift.

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
