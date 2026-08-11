#define _POSIX_C_SOURCE 200809L

#include "e2.h"
#include "observation.h"
#include "sha256.h"

#include <errno.h>
#include <inttypes.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

#define TW_PATH_BYTES 4096U

static int fail(const char *message) {
    (void)fprintf(stderr, "tw-verify-result-c: %s\n", message);
    return 1;
}

static bool path_for(char out[TW_PATH_BYTES], const char *dir, const char *name) {
    const int written = snprintf(out, TW_PATH_BYTES, "%s/%s", dir, name);
    return written >= 0 && (size_t)written < TW_PATH_BYTES;
}

static bool read_regular_file(const char *path, size_t maximum, uint8_t **data, size_t *length) {
    struct stat status;
    if (lstat(path, &status) != 0 || !S_ISREG(status.st_mode) || status.st_size <= 0 || (uintmax_t)status.st_size > (uintmax_t)maximum) return false;
    FILE *file = fopen(path, "rb");
    if (file == NULL) return false;
    const size_t size = (size_t)status.st_size;
    uint8_t *buffer = malloc(size + 1U);
    if (buffer == NULL) { (void)fclose(file); return false; }
    const size_t count = fread(buffer, 1U, size + 1U, file);
    const int close_result = fclose(file);
    if (count != size || close_result != 0) { free(buffer); return false; }
    *data = buffer;
    *length = size;
    return true;
}

static void digest(const uint8_t *data, size_t length, uint8_t out[32U]) {
    tw_sha256_ctx context;
    tw_sha256_init(&context);
    tw_sha256_update(&context, data, length);
    tw_sha256_final(&context, out);
}

static bool load_entry(const char *dir, const tw_e2_entry *entry, uint8_t **data, size_t *length) {
    char path[TW_PATH_BYTES];
    if (entry == NULL || !path_for(path, dir, entry->name) || !read_regular_file(path, TW_E2_MAX_ARTIFACT_BYTES, data, length)) return false;
    if ((uint64_t)*length != entry->size) { free(*data); *data = NULL; return false; }
    uint8_t actual[32U];
    digest(*data, *length, actual);
    if (memcmp(actual, entry->digest, 32U) != 0) { free(*data); *data = NULL; return false; }
    return true;
}

static bool digest_bound(const tw_e2_manifest *manifest, const char *name, const uint8_t expected[32U]) {
    const tw_e2_entry *entry = tw_e2_find_entry(manifest, name);
    return entry != NULL && memcmp(entry->digest, expected, 32U) == 0;
}

int main(int argc, char **argv) {
    if (argc != 2) {
        (void)fprintf(stderr, "usage: tw-verify-result-c BUNDLE_DIR\n");
        return 2;
    }
    char path[TW_PATH_BYTES];
    if (!path_for(path, argv[1], "manifest.cbor")) return fail("bundle path is too long");
    uint8_t *manifest_bytes = NULL;
    size_t manifest_length = 0U;
    if (!read_regular_file(path, TW_E2_MAX_MANIFEST_BYTES, &manifest_bytes, &manifest_length)) return fail("final manifest is missing, non-regular, or outside bounds");
    tw_e2_manifest manifest;
    memset(&manifest, 0, sizeof(manifest));
    if (!tw_e2_parse_manifest(manifest_bytes, manifest_length, &manifest)) { free(manifest_bytes); return fail("invalid or non-canonical manifest"); }
    free(manifest_bytes);

    for (size_t i = 0U; i < manifest.entry_count; ++i) {
        uint8_t *artifact = NULL;
        size_t artifact_length = 0U;
        if (!load_entry(argv[1], &manifest.entries[i], &artifact, &artifact_length)) { free(artifact); return fail("artifact is missing, non-regular, oversized, or has the wrong digest"); }
        free(artifact);
    }

    uint8_t *result_bytes = NULL;
    size_t result_length = 0U;
    if (!load_entry(argv[1], tw_e2_find_entry(&manifest, "result.cbor"), &result_bytes, &result_length)) { free(result_bytes); return fail("unable to load result"); }
    tw_e2_result result;
    memset(&result, 0, sizeof(result));
    if (!tw_e2_parse_result(result_bytes, result_length, &result)) { free(result_bytes); return fail("invalid or non-canonical result"); }
    uint8_t result_digest[32U];
    digest(result_bytes, result_length, result_digest);
    free(result_bytes);
    if (memcmp(result_digest, manifest.result_digest, 32U) != 0 ||
        !digest_bound(&manifest, "input.cbor", result.input_digest) || !digest_bound(&manifest, "observation.cbor", result.observation_digest) ||
        !digest_bound(&manifest, "transport.cbor", result.transport_digest) || !digest_bound(&manifest, "adapter.cbor", result.adapter_digest) ||
        !digest_bound(&manifest, "contract.cbor", result.contract_digest) || !digest_bound(&manifest, "semantic-closure.cbor", result.semantic_closure_digest)) return fail("result digest binding mismatch");

    uint8_t *observation_bytes = NULL;
    size_t observation_length = 0U;
    if (!load_entry(argv[1], tw_e2_find_entry(&manifest, "observation.cbor"), &observation_bytes, &observation_length)) { free(observation_bytes); return fail("unable to load observation"); }
    tw_observation observation;
    memset(&observation, 0, sizeof(observation));
    if (!tw_parse_observation(observation_bytes, observation_length, &observation) || strcmp(observation.retrieved_at, result.observed_at) != 0) { free(observation_bytes); return fail("invalid observation or retrieval-time binding"); }
    free(observation_bytes);
    const tw_e2_entry *body = tw_e2_find_entry(&manifest, "representation.body");
    if (body == NULL || body->size != observation.body_size || memcmp(body->digest, observation.body_hash, 32U) != 0) return fail("representation does not match observation");

    uint8_t *closure_bytes = NULL;
    size_t closure_length = 0U;
    if (!load_entry(argv[1], tw_e2_find_entry(&manifest, "semantic-closure.cbor"), &closure_bytes, &closure_length)) { free(closure_bytes); return fail("unable to load semantic closure"); }
    const bool closure_valid = tw_e2_parse_semantic_closure(closure_bytes, closure_length);
    free(closure_bytes);
    if (!closure_valid) return fail("invalid or non-canonical semantic closure");

    (void)printf("{\"status\":\"verified\",\"format\":\"tw.result/0.2\",\"origin_id\":\"%s\",\"operation_id\":\"%s\",\"field_count\":%zu,\"artifacts\":%zu}\n",
                 result.origin_id, result.operation_id, result.field_count, manifest.entry_count);
    return 0;
}
