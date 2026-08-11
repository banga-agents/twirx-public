# TWABI — design direction

TWABI will provide a bounded adapter boundary suitable for WebAssembly guests written in C, Go, Zig, AssemblyScript, or other languages.

The first intended contract is canonical observation input and extraction-candidate output with no ambient network, filesystem, environment, secrets, clock, or randomness.

No TWABI runtime is implemented in Genesis Gate 1.
