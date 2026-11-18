package mount

import "syscall"

// noAttribute is the errno returned when an extended attribute does not exist.
const noAttribute = syscall.ENOATTR
