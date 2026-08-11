#include "e2.h"

#include <stddef.h>
#include <stdint.h>
#include <string.h>

int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
    tw_e2_result result;
    memset(&result, 0, sizeof(result));
    (void)tw_e2_parse_result(data, size, &result);
    tw_e2_manifest manifest;
    memset(&manifest, 0, sizeof(manifest));
    (void)tw_e2_parse_manifest(data, size, &manifest);
    (void)tw_e2_parse_semantic_closure(data, size);
    return 0;
}
