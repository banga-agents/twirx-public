#include "observation.h"

#include <stddef.h>
#include <stdint.h>
#include <string.h>

int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
    tw_observation observation;
    memset(&observation, 0, sizeof(observation));
    (void)tw_parse_observation(data, size, &observation);
    return 0;
}
