#ifndef TW_SHA256_H
#define TW_SHA256_H

#include <stddef.h>
#include <stdint.h>

typedef struct tw_sha256_ctx {
    uint32_t state[8];
    uint64_t bit_count;
    uint8_t buffer[64];
    size_t buffer_len;
} tw_sha256_ctx;

void tw_sha256_init(tw_sha256_ctx *ctx);
void tw_sha256_update(tw_sha256_ctx *ctx, const uint8_t *data, size_t len);
void tw_sha256_final(tw_sha256_ctx *ctx, uint8_t out[32]);

#endif
