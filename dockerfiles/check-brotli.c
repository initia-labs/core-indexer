#include <dlfcn.h>
#include <stdint.h>
#include <stdio.h>

#include "common/version.h"

int main(void) {
    /* Resolve through libvips so this checks its actual decoder dependency. */
    void *vips = dlopen("libvips.so.42", RTLD_NOW | RTLD_LOCAL);
    if (vips == NULL) {
        fprintf(stderr, "Cannot load libvips: %s\n", dlerror());
        return 1;
    }

    uint32_t (*decoder_version)(void) = dlsym(vips, "BrotliDecoderVersion");
    if (decoder_version == NULL) {
        fprintf(stderr, "Cannot resolve libvips' Brotli decoder: %s\n", dlerror());
        dlclose(vips);
        return 1;
    }

    uint32_t actual = decoder_version();
    if (actual != BROTLI_VERSION) {
        fprintf(stderr, "Unexpected Brotli decoder: 0x%x; expected 0x%x\n",
                actual, BROTLI_VERSION);
        dlclose(vips);
        return 1;
    }

    printf("libvips loads patched Brotli %u.%u.%u\n",
           actual >> 24, (actual >> 12) & 0xfff, actual & 0xfff);
    dlclose(vips);
    return 0;
}
