#include <dlfcn.h>
#include <sys/mman.h>

int safe_add(int a, int b) { return a + b; }

void *lookup(const char *name) {
    void *lib = dlopen("libm.so", 0);
    void *sym = dlsym(lib, name);
    return sym;
}

void *map_region(unsigned long size) {
    return mmap(0, size, 0, 0, -1, 0);
}
