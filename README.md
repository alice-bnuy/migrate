# Setup Backup Tool

Ferramenta CLI para backup e restauração de arquivos de configuração do sistema.

## Dependências

- Go >= 1.20
- Nushell

### Instalação do Nushell (Fedora)

```sh
sudo dnf install nushell
```

### Instalação do Nushell (Ubuntu/Debian)

```sh
sudo apt install nushell
```

Ou, para qualquer distribuição Linux, via GitHub Releases:

```sh
curl -sSL https://github.com/nushell/nushell/releases/latest/download/nu-linux-x86_64.tar.gz -o nu.tar.gz
tar -xzf nu.tar.gz
sudo mv nu*/nu /usr/local/bin/nu
rm -rf nu*
```

## Build

```sh
make build
```

## Uso

```sh
tools/setup create   # Cria backup dos arquivos do sistema em assets/files
tools/setup apply    # Aplica backup de assets/files para o sistema
```
