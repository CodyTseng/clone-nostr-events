# clone-nostr-events

Clone your Nostr events with a new private key when your private key is leaked.

## Build

Build clone-nostr-events by running (from the root of the repository):

```bash
go mod download # download dependencies
go build
```

## Usage

```bash
./clone-nostr-events <npub> <initial relay> <new nsec>
```

- `npub`: the npub of the account you want to clone.
- `initial relay`: the relay (need to have your relay list metadata) you want to start cloning from.
- `new nsec`: the nsec of the account you want to clone to.

## Example

```bash
./clone-nostr-events npub15zt9nnv7azwdxapmc2d2vlklr47397mzg6vle57k6vlw7fgtq8ns8yprpd wss://nostr-relay.app nsec192kvunzhmtl2rfrzg3736lmedhcxjv8ar4d0tnqs0lgafrf9z2ns6lt98t
```

## Donations

If you like this project, you can buy me a coffee ☕️ ⚡ codytseng@getalby.com ⚡

## License

This project is MIT licensed.