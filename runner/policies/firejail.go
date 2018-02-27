package policies

// This templates generates a sandbox policy file suitable for
// running relatively-untrusted apps via itch.
//
// TODO: figure a better way — blacklists aren't so good.
// whitelist doesn't seem to work with exclusions, though?

const FirejailTemplate = `
blacklist ~/.config/chromium
blacklist ~/.config/chrome
blacklist ~/.mozilla
`
