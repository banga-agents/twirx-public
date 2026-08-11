#include "observation.h"
#include "sha256.h"

#include <errno.h>
#include <inttypes.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define TW_PATH_BYTES 4096U

static int fail(const char *message) {
    (void)fprintf(stderr, "tw-verify-c: %s\n", message);
    return 1;
}

static bool read_bounded_file(const char *path, size_t maximum, uint8_t **data, size_t *length) {
    FILE *file = fopen(path, "rb");
    if (file == NULL) {
        return false;
    }
    if (fseek(file, 0L, SEEK_END) != 0) {
        (void)fclose(file);
        return false;
    }
    const long signed_size = ftell(file);
    if (signed_size <= 0L || (uintmax_t)signed_size > (uintmax_t)maximum) {
        (void)fclose(file);
        return false;
    }
    if (fseek(file, 0L, SEEK_SET) != 0) {
        (void)fclose(file);
        return false;
    }
    const size_t size = (size_t)signed_size;
    uint8_t *buffer = malloc(size);
    if (buffer == NULL) {
        (void)fclose(file);
        return false;
    }
    const size_t read_count = fread(buffer, 1U, size, file);
    const int close_result = fclose(file);
    if (read_count != size || close_result != 0) {
        free(buffer);
        return false;
    }
    *data = buffer;
    *length = size;
    return true;
}

static void hash_hex(const uint8_t hash[32], char out[65]) {
    static const char hex[] = "0123456789abcdef";
    for (size_t i = 0U; i < 32U; ++i) {
        out[i * 2U] = hex[hash[i] >> 4U];
        out[i * 2U + 1U] = hex[hash[i] & UINT8_C(0x0f)];
    }
    out[64] = '\0';
}

static bool verify_body(const char *cas_root, const tw_observation *observation, char digest_hex[65]) {
    hash_hex(observation->body_hash, digest_hex);
    char path[TW_PATH_BYTES];
    const int written = snprintf(path, sizeof(path), "%s/sha256/%.2s/%.2s/%s", cas_root, digest_hex, digest_hex + 2, digest_hex);
    if (written < 0 || (size_t)written >= sizeof(path)) {
        return false;
    }
    FILE *file = fopen(path, "rb");
    if (file == NULL) {
        return false;
    }
    tw_sha256_ctx ctx;
    tw_sha256_init(&ctx);
    uint8_t buffer[32768U];
    uint64_t total = UINT64_C(0);
    for (;;) {
        const size_t count = fread(buffer, 1U, sizeof(buffer), file);
        if (count > 0U) {
            if (UINT64_MAX - total < (uint64_t)count) {
                (void)fclose(file);
                return false;
            }
            total += (uint64_t)count;
            if (total > observation->body_size) {
                (void)fclose(file);
                return false;
            }
            tw_sha256_update(&ctx, buffer, count);
        }
        if (count < sizeof(buffer)) {
            if (ferror(file) != 0) {
                (void)fclose(file);
                return false;
            }
            break;
        }
    }
    if (fclose(file) != 0 || total != observation->body_size) {
        return false;
    }
    uint8_t actual[32U];
    tw_sha256_final(&ctx, actual);
    return memcmp(actual, observation->body_hash, sizeof(actual)) == 0;
}

int main(int argc, char **argv) {
    if (argc != 3) {
        (void)fprintf(stderr, "usage: tw-verify-c OBSERVATION.cbor CAS_ROOT\n");
        return 2;
    }
    uint8_t *envelope = NULL;
    size_t envelope_length = 0U;
    if (!read_bounded_file(argv[1], TW_MAX_ENVELOPE_BYTES, &envelope, &envelope_length)) {
        (void)fprintf(stderr, "tw-verify-c: unable to read bounded envelope %s: %s\n", argv[1], strerror(errno));
        return 1;
    }
    tw_observation observation;
    memset(&observation, 0, sizeof(observation));
    if (!tw_parse_observation(envelope, envelope_length, &observation)) {
        free(envelope);
        return fail("invalid or non-canonical observation envelope");
    }
    free(envelope);

    char digest_hex[65U];
    if (!verify_body(argv[2], &observation, digest_hex)) {
        return fail("body is missing, oversized, or does not match the declared digest");
    }
    (void)printf("{\"status\":\"verified\",\"version\":%" PRIu64 ",\"body_digest\":\"sha256:%s\",\"body_size\":%" PRIu64 "}\n",
                 observation.version, digest_hex, observation.body_size);
    return 0;
}
