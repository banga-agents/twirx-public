#ifndef TW_OBSERVATION_H
#define TW_OBSERVATION_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#define TW_MAX_ENVELOPE_BYTES (64U * 1024U)
#define TW_MAX_URL_BYTES 8192U
#define TW_MAX_MEDIA_TYPE_BYTES 256U
#define TW_MAX_RETRIEVED_AT_BYTES 30U
#define TW_MAX_ID_BYTES 256U
#define TW_MAX_BODY_BYTES UINT64_C(2097152)

typedef struct tw_observation {
    uint64_t version;
    char request_url[TW_MAX_URL_BYTES + 1U];
    char final_url[TW_MAX_URL_BYTES + 1U];
    char method[4U];
    uint64_t status;
    char media_type[TW_MAX_MEDIA_TYPE_BYTES + 1U];
    char retrieved_at[TW_MAX_RETRIEVED_AT_BYTES + 1U];
    uint8_t body_hash[32U];
    uint64_t body_size;
    char policy_id[TW_MAX_ID_BYTES + 1U];
    char observer_id[TW_MAX_ID_BYTES + 1U];
} tw_observation;

bool tw_parse_observation(const uint8_t *data, size_t length, tw_observation *observation);

#endif
