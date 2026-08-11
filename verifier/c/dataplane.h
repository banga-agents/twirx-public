#ifndef TW_DATAPLANE_H
#define TW_DATAPLANE_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#define TW_DP_MAX_DOCUMENT_BYTES (4U * 1024U * 1024U)

bool tw_dp_verify(const char *kind, const uint8_t *data, size_t length);

#endif
