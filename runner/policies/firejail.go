package policies

// This templates generates a sandbox policy file suitable for
// running relatively-untrusted apps via itch.
//
// TODO: figure a better way — blacklists aren't so good.
// whitelist doesn't seem to work with exclusions, though?

const FirejailTemplate = `
caps.drop all
# ipc-namespace
netfilter
# no3d
# nodvd
# nogroups
nonewprivs
noroot
# nosound
# notv
# novideo
protocol unix,inet,inet6
seccomp
# shell none

# disable-mnt
# private
# private-bin program
# private-dev
# private-etc none
# private-lib
# private-tmp

# memory-deny-write-execute
# noexec ${HOME}
# noexec /tmp
`
