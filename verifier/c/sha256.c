#include "sha256.h"

#include <string.h>

static uint32_t rotr32(uint32_t value, unsigned int shift) {
    return (value >> shift) | (value << (32U - shift));
}

static uint32_t choose(uint32_t x, uint32_t y, uint32_t z) {
    return (x & y) ^ ((~x) & z);
}

static uint32_t majority(uint32_t x, uint32_t y, uint32_t z) {
    return (x & y) ^ (x & z) ^ (y & z);
}

static uint32_t big_sigma0(uint32_t x) {
    return rotr32(x, 2U) ^ rotr32(x, 13U) ^ rotr32(x, 22U);
}

static uint32_t big_sigma1(uint32_t x) {
    return rotr32(x, 6U) ^ rotr32(x, 11U) ^ rotr32(x, 25U);
}

static uint32_t small_sigma0(uint32_t x) {
    return rotr32(x, 7U) ^ rotr32(x, 18U) ^ (x >> 3U);
}

static uint32_t small_sigma1(uint32_t x) {
    return rotr32(x, 17U) ^ rotr32(x, 19U) ^ (x >> 10U);
}

static uint32_t load_be32(const uint8_t src[4]) {
    return ((uint32_t)src[0] << 24U) |
           ((uint32_t)src[1] << 16U) |
           ((uint32_t)src[2] << 8U) |
           (uint32_t)src[3];
}

static void store_be64(uint8_t dst[8], uint64_t value) {
    for (size_t i = 0U; i < 8U; ++i) {
        dst[7U - i] = (uint8_t)(value & UINT64_C(0xff));
        value >>= 8U;
    }
}

static void transform(tw_sha256_ctx *ctx, const uint8_t block[64]) {
    static const uint32_t k[64] = {
        UINT32_C(0x428a2f98), UINT32_C(0x71374491), UINT32_C(0xb5c0fbcf), UINT32_C(0xe9b5dba5),
        UINT32_C(0x3956c25b), UINT32_C(0x59f111f1), UINT32_C(0x923f82a4), UINT32_C(0xab1c5ed5),
        UINT32_C(0xd807aa98), UINT32_C(0x12835b01), UINT32_C(0x243185be), UINT32_C(0x550c7dc3),
        UINT32_C(0x72be5d74), UINT32_C(0x80deb1fe), UINT32_C(0x9bdc06a7), UINT32_C(0xc19bf174),
        UINT32_C(0xe49b69c1), UINT32_C(0xefbe4786), UINT32_C(0x0fc19dc6), UINT32_C(0x240ca1cc),
        UINT32_C(0x2de92c6f), UINT32_C(0x4a7484aa), UINT32_C(0x5cb0a9dc), UINT32_C(0x76f988da),
        UINT32_C(0x983e5152), UINT32_C(0xa831c66d), UINT32_C(0xb00327c8), UINT32_C(0xbf597fc7),
        UINT32_C(0xc6e00bf3), UINT32_C(0xd5a79147), UINT32_C(0x06ca6351), UINT32_C(0x14292967),
        UINT32_C(0x27b70a85), UINT32_C(0x2e1b2138), UINT32_C(0x4d2c6dfc), UINT32_C(0x53380d13),
        UINT32_C(0x650a7354), UINT32_C(0x766a0abb), UINT32_C(0x81c2c92e), UINT32_C(0x92722c85),
        UINT32_C(0xa2bfe8a1), UINT32_C(0xa81a664b), UINT32_C(0xc24b8b70), UINT32_C(0xc76c51a3),
        UINT32_C(0xd192e819), UINT32_C(0xd6990624), UINT32_C(0xf40e3585), UINT32_C(0x106aa070),
        UINT32_C(0x19a4c116), UINT32_C(0x1e376c08), UINT32_C(0x2748774c), UINT32_C(0x34b0bcb5),
        UINT32_C(0x391c0cb3), UINT32_C(0x4ed8aa4a), UINT32_C(0x5b9cca4f), UINT32_C(0x682e6ff3),
        UINT32_C(0x748f82ee), UINT32_C(0x78a5636f), UINT32_C(0x84c87814), UINT32_C(0x8cc70208),
        UINT32_C(0x90befffa), UINT32_C(0xa4506ceb), UINT32_C(0xbef9a3f7), UINT32_C(0xc67178f2)
    };

    uint32_t w[64];
    for (size_t i = 0U; i < 16U; ++i) {
        w[i] = load_be32(&block[i * 4U]);
    }
    for (size_t i = 16U; i < 64U; ++i) {
        w[i] = small_sigma1(w[i - 2U]) + w[i - 7U] + small_sigma0(w[i - 15U]) + w[i - 16U];
    }

    uint32_t a = ctx->state[0];
    uint32_t b = ctx->state[1];
    uint32_t c = ctx->state[2];
    uint32_t d = ctx->state[3];
    uint32_t e = ctx->state[4];
    uint32_t f = ctx->state[5];
    uint32_t g = ctx->state[6];
    uint32_t h = ctx->state[7];

    for (size_t i = 0U; i < 64U; ++i) {
        const uint32_t t1 = h + big_sigma1(e) + choose(e, f, g) + k[i] + w[i];
        const uint32_t t2 = big_sigma0(a) + majority(a, b, c);
        h = g;
        g = f;
        f = e;
        e = d + t1;
        d = c;
        c = b;
        b = a;
        a = t1 + t2;
    }

    ctx->state[0] += a;
    ctx->state[1] += b;
    ctx->state[2] += c;
    ctx->state[3] += d;
    ctx->state[4] += e;
    ctx->state[5] += f;
    ctx->state[6] += g;
    ctx->state[7] += h;
}

void tw_sha256_init(tw_sha256_ctx *ctx) {
    static const uint32_t initial[8] = {
        UINT32_C(0x6a09e667), UINT32_C(0xbb67ae85), UINT32_C(0x3c6ef372), UINT32_C(0xa54ff53a),
        UINT32_C(0x510e527f), UINT32_C(0x9b05688c), UINT32_C(0x1f83d9ab), UINT32_C(0x5be0cd19)
    };
    memcpy(ctx->state, initial, sizeof(initial));
    ctx->bit_count = UINT64_C(0);
    ctx->buffer_len = 0U;
    memset(ctx->buffer, 0, sizeof(ctx->buffer));
}

void tw_sha256_update(tw_sha256_ctx *ctx, const uint8_t *data, size_t len) {
    if (len == 0U) {
        return;
    }
    ctx->bit_count += (uint64_t)len * UINT64_C(8);
    while (len > 0U) {
        const size_t available = sizeof(ctx->buffer) - ctx->buffer_len;
        const size_t take = len < available ? len : available;
        memcpy(&ctx->buffer[ctx->buffer_len], data, take);
        ctx->buffer_len += take;
        data += take;
        len -= take;
        if (ctx->buffer_len == sizeof(ctx->buffer)) {
            transform(ctx, ctx->buffer);
            ctx->buffer_len = 0U;
        }
    }
}

void tw_sha256_final(tw_sha256_ctx *ctx, uint8_t out[32]) {
    ctx->buffer[ctx->buffer_len++] = UINT8_C(0x80);
    if (ctx->buffer_len > 56U) {
        memset(&ctx->buffer[ctx->buffer_len], 0, sizeof(ctx->buffer) - ctx->buffer_len);
        transform(ctx, ctx->buffer);
        ctx->buffer_len = 0U;
    }
    memset(&ctx->buffer[ctx->buffer_len], 0, 56U - ctx->buffer_len);
    store_be64(&ctx->buffer[56], ctx->bit_count);
    transform(ctx, ctx->buffer);

    for (size_t i = 0U; i < 8U; ++i) {
        out[i * 4U] = (uint8_t)(ctx->state[i] >> 24U);
        out[i * 4U + 1U] = (uint8_t)(ctx->state[i] >> 16U);
        out[i * 4U + 2U] = (uint8_t)(ctx->state[i] >> 8U);
        out[i * 4U + 3U] = (uint8_t)ctx->state[i];
    }
    memset(ctx, 0, sizeof(*ctx));
}
