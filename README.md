# CRV Go

Aplicativo local e offline para registrar sessões inspiradas nas etapas I–III de CRV e escolher uma entre quatro imagens após bloquear o registro. Ele não interpreta desenhos nem estabelece um mecanismo paranormal. O histórico fica no dispositivo e não é sincronizado.

## Usar uma distribuição

Os pacotes Windows e Linux ficam em `packages/` depois de executar `npm run package`. Extraia o arquivo da plataforma e mantenha `farsight/` ao lado do executável. No Windows, abra `crv.exe`. No Linux, execute `chmod +x crv` e depois `./crv`. O aplicativo abre a interface no navegador padrão; não requer Node, Go ou internet para funcionar. A primeira execução valida e instala o catálogo nos dados locais do usuário.

Para encerrar, use **Salvar e encerrar**. Para levar dados a outra instalação, use Configurações para criar e restaurar um backup ZIP. Históricos pertencem a cada instalação.

Cada distribuição contém `README.txt`, `THIRD_PARTY_NOTICES.txt` e os textos de licença em `LICENSES/`. O pacote contém o catálogo Farsight e créditos de origem; consulte [os avisos de terceiros](docs/THIRD_PARTY_NOTICES.md) para fontes, limites e hashes específicos do build. O hash dos arquivos de distribuição está em `packages/SHA256SUMS.txt`.

## Desenvolvimento

Requer Node.js 24+ e Go 1.26.5+. No PowerShell, use `npm.cmd` caso `npm` invoque um script bloqueado.

```sh
npm ci
npm run check
npm run build
go test ./...
go vet ./...
npm run test:e2e
npm run package
```

`npm run package` recompila a interface, compila `crv.exe` para Windows amd64 e `crv` para Linux amd64 com `CGO_ENABLED=0`, e grava os arquivos `.tar.gz` em `packages/`. É uma distribuição local de desenvolvimento, sem instalador, assinatura ou publicação. Compilar o binário Linux no Windows não valida sua execução em Linux.

Execução de desenvolvimento isolada, sem usar o diretório de dados normal:

```sh
go run . --data-dir tmp-data/demo --catalog-dir farsight --no-browser
```

O servidor imprime a URL local de inicialização. Sem flags, o executável usa `farsight/` ao lado do binário e os dados locais do usuário.

Para testar um pacote extraído pelo navegador automatizado no PowerShell:

```powershell
$env:CRV_PACKAGE_DIR = 'D:\caminho do pacote\crv-go-windows-amd64'
npm.cmd run test:e2e
```

Sem essa variável, o teste de distribuição é ignorado. Consulte [a verificação](docs/VERIFICACAO.md) para evidência e pendências: a execução Linux e os testes nativos/offline manuais ainda estão pendentes.

## Protótipo histórico

`crv_prototipo.html` é um protótipo independente, abre diretamente no navegador e mantém demonstrações em localStorage. Seus registros, imagens e resultados não são dados experimentais e não representam a aplicação final. `dist/index.html` é a interface do executável, gerada pelo Vite e embutida no binário; não é uma versão independente do servidor Go.

O aplicativo final persiste sessões em SQLite e mantém o alvo no backend. As quatro imagens de cada sessão são distintas, mas podem se repetir em sessões novas. A fase de implementação inclui histórico, estatísticas, catálogo, CSV, impressão para Save as PDF e backup/restauração ZIP.

## Créditos

O catálogo versionado em `farsight/` conserva os créditos originais e sua declaração “Local research archive only.” As fotografias são do Farsight Institute / Courtney Brown e de fotógrafos terceiros, conforme os metadados preservados; isso não concede direitos adicionais sobre os arquivos. A fonte Inter usa SIL Open Font License 1.1. Os detalhes ficam em `docs/THIRD_PARTY_NOTICES.md`.
