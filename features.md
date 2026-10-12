# Features

Registro das features entregues, da mais nova para a mais antiga. Adicione uma entrada a cada commit.

## 2026-10-12 11:14 — Site e documentação em dia

- Resolve as pendências de HTML das cores por alias, do alias padrão e do modo pipe.
- **Site (`docs/index.html`):** o mockup da tela mostra os aliases nas cores reais (calculadas pelo mesmo hash do cliente) e o dono da tela em negrito. Destaque novo "Cada pessoa com sua cor". O card de pipes perdeu o "em breve" e ganhou um exemplo; "Comece em um minuto" mostra o `uptime | teachat`; o modo pipe saiu de "O que vem aí".
- **Documentação (`docs/docs.html`):**
  - Referência do cliente: `--alias` com padrão `$USER`, cores na área de mensagens, códigos de saída do modo pipe e uma subseção nova **Modo pipe** (uso, eco, uma mensagem por linha, sem limite de tamanho, seguro em scripts). Link na barra lateral.
  - Cliente por dentro: subseções novas **Cores por alias** (hash, negrito, `tview.Escape`, a paleta) e **Modo pipe por dentro** (as duas goroutines, por que ler a sala, o close no fim, o teste da sala inundada). O código mostrado foi regenerado a partir dos arquivos atuais, com o `pipe.go` inteiro.
  - Arquitetura: `pipe.go` no mapa do repositório, `golang.org/x/term` nas dependências, contagem de linhas atualizada (~380).
  - Limitações: sai "pipe sem terminal falha", entra "logs grandes inundam a sala"; "cliente lento" agora com o número medido (2082 de 5000).

## 2026-10-12 10:58 — Modo pipe

- Quando o stdin não é um terminal (`comando | teachat`), o cliente não abre a tela: envia cada linha para a sala, repete a linha no terminal (como o `tee`) e sai quando a entrada acaba. Código em `cmd/client/pipe.go`; o `main.go` só ganhou o `term.IsTerminal`.
- `tail -f app.log | teachat --room ops` funciona como transmissão contínua, até o Ctrl+C.
- Linha vazia aparece no eco, mas não vai para a sala. Linha sem `\n` no fim também é enviada.
- Sem limite de tamanho de linha: `bufio.Reader.ReadString` em vez do `bufio.Scanner`, que desiste em 64 KB.
- **Saída limpa:** no fim, manda o frame de close do WebSocket e espera o servidor responder (até 2 s), para as últimas linhas não se perderem. Código de saída 0; falha ao conectar ou conexão caindo no meio: 1, com o erro no stderr.
- **Por que lê a sala mesmo sem mostrar nada:** as mensagens dos outros continuam chegando. Sem ler, os buffers de rede enchem e o broadcast do servidor trava a sala inteira esperando pelo pipe. Uma goroutine lê e descarta tudo.
- **Testado:** 2000 linhas (`seq`) chegam todas, inclusive a última; uma linha de 200 KB chega inteira; servidor fora do ar dá `connection refused` e código 1. Com uma sala inundada por 5000 mensagens de 2 KB e um pipe lento conectado, todas as 5000 chegaram aos outros. Com um cliente que nunca lê no lugar do pipe, a sala travou em 2082 de 5000: é o limite "cliente lento atrasa a sala", agora medido.
- `golang.org/x/term` passou de dependência indireta para direta (já estava no projeto).
- **Docs (HTML):** atualizados em 2026-10-12.

## 2026-10-12 10:06 — Alias padrão é o seu usuário

- Sem `--alias`, o cliente usa o seu usuário do sistema (`$USER`) em vez de `anonimo`. Quem não tem `$USER` definido (cron, alguns containers) continua como `anonimo`.
- **Por quê:** quase ninguém precisa mais digitar `--alias`. Uma linha (`cmp.Or`), sem dependência.
- **Docs (HTML):** atualizados em 2026-10-12.

## 2026-10-12 09:50 — Cores por alias

- Na área de mensagens, o alias de cada pessoa aparece numa cor própria, escolhida pelo hash do alias (FNV-32a) numa paleta de 12 cores. O mesmo alias tem sempre a mesma cor, em qualquer cliente, sem mudar o servidor nem o protocolo.
- Suas próprias mensagens: alias em negrito, na cor normal do texto. Vale só para as linhas que você enviou, não para o alias: se outra pessoa entra com o seu alias, ela aparece colorida, não em negrito.
- Só o alias é colorido; o texto da mensagem continua na cor normal.
- **Segurança:** para colorir, o tview passou a interpretar tags como `[red]`. Tudo que vem de fora (alias, texto, nome da sala no título) passa por `tview.Escape`, então quem manda `[red]oi` continua vendo e mostrando `[red]oi` literal.
- **Limites:** com 12 cores, colisões acontecem (5 pessoas numa sala: ~60% de chance de duas terem a mesma cor). O cliente acha o alias cortando a linha no primeiro `": "`, porque o servidor manda `alias: texto` como string única; um alias com `": "` dentro é cortado errado. Os dois somem quando as mensagens virarem estruturadas (JSON) no protocolo.
- Testado ao vivo com 5 clientes na mesma sala: cores diferentes por alias, negrito só nas próprias mensagens, `[red]oi` aparecendo literal.
- **Docs (HTML):** atualizados em 2026-10-12.

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
