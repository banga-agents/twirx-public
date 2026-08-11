#ifndef TW_E2_H
#define TW_E2_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#define TW_E2_MAX_RESULT_BYTES (1024U * 1024U)
#define TW_E2_MAX_MANIFEST_BYTES (64U * 1024U)
#define TW_E2_MAX_ARTIFACT_BYTES (4U * 1024U * 1024U)
#define TW_E2_MAX_ARTIFACTS 32U
#define TW_E2_MAX_FIELDS 128U
#define TW_E2_MAX_NAME_BYTES 255U
#define TW_E2_MAX_TEXT_BYTES (16U * 1024U)

typedef struct tw_e2_entry {
    char name[TW_E2_MAX_NAME_BYTES + 1U];
    uint8_t digest[32U];
    uint64_t size;
} tw_e2_entry;

typedef struct tw_e2_manifest {
    uint8_t result_digest[32U];
    size_t entry_count;
    tw_e2_entry entries[TW_E2_MAX_ARTIFACTS];
} tw_e2_manifest;

typedef struct tw_e2_result {
    char invocation_id[TW_E2_MAX_TEXT_BYTES + 1U];
    char origin_id[TW_E2_MAX_TEXT_BYTES + 1U];
    char origin_version[TW_E2_MAX_TEXT_BYTES + 1U];
    char operation_id[TW_E2_MAX_TEXT_BYTES + 1U];
    char operation_version[TW_E2_MAX_TEXT_BYTES + 1U];
    char observed_at[64U];
    uint8_t input_digest[32U];
    uint8_t observation_digest[32U];
    uint8_t transport_digest[32U];
    uint8_t adapter_digest[32U];
    uint8_t contract_digest[32U];
    uint8_t semantic_closure_digest[32U];
    size_t field_count;
} tw_e2_result;

bool tw_e2_parse_result(const uint8_t *data, size_t length, tw_e2_result *result);
bool tw_e2_parse_manifest(const uint8_t *data, size_t length, tw_e2_manifest *manifest);
bool tw_e2_parse_semantic_closure(const uint8_t *data, size_t length);
const tw_e2_entry *tw_e2_find_entry(const tw_e2_manifest *manifest, const char *name);

#endif
