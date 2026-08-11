#include "e2.h"

#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static bool read_file(const char *path, uint8_t **data, size_t *length) {
    FILE *file = fopen(path, "rb");
    if (file == NULL) return false;
    if (fseek(file, 0L, SEEK_END) != 0) { (void)fclose(file); return false; }
    const long size = ftell(file);
    if (size <= 0L || (unsigned long)size > (unsigned long)TW_E2_MAX_ARTIFACT_BYTES || fseek(file, 0L, SEEK_SET) != 0) { (void)fclose(file); return false; }
    *data = malloc((size_t)size);
    if (*data == NULL) { (void)fclose(file); return false; }
    *length = (size_t)size;
    const bool complete = fread(*data, 1U, *length, file) == *length;
    const bool closed = fclose(file) == 0;
    const bool ok = complete && closed;
    if (!ok) { free(*data); *data = NULL; }
    return ok;
}

int main(int argc, char **argv) {
    if (argc != 3) {
        (void)fprintf(stderr, "usage: tw-verify-e2-artifact-c result|manifest|closure FILE\n");
        return 2;
    }
    uint8_t *data = NULL;
    size_t length = 0U;
    if (!read_file(argv[2], &data, &length)) return 1;
    bool valid = false;
    if (strcmp(argv[1], "result") == 0) {
        tw_e2_result value;
        memset(&value, 0, sizeof(value));
        valid = tw_e2_parse_result(data, length, &value);
    } else if (strcmp(argv[1], "manifest") == 0) {
        tw_e2_manifest value;
        memset(&value, 0, sizeof(value));
        valid = tw_e2_parse_manifest(data, length, &value);
    } else if (strcmp(argv[1], "closure") == 0) {
        valid = tw_e2_parse_semantic_closure(data, length);
    }
    free(data);
    return valid ? 0 : 1;
}
