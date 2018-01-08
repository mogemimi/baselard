#pragma once

#ifdef __APPLE_CC__
#include <TargetConditionals.h>
#endif

#if defined(linux) || defined(__linux) || defined(__linux__)
    // Linux
    #define ENGINE_PLATFORM_LINUX
#elif defined(__FreeBSD__) || defined(__NetBSD__) || defined(__OpenBSD__)
    // BSD
    #define ENGINE_PLATFORM_BSD
#elif defined(_WIN32) || defined(__WIN32__) || defined(WIN32)
    // Windows
    #define ENGINE_PLATFORM_WINDOWS
#elif defined(ANDROID) || defined(__ANDROID__)
    // Android OS
    #define ENGINE_PLATFORM_ANDROID
#elif defined(__APPLE_CC__) && defined(TARGET_OS_IPHONE) && TARGET_OS_IPHONE
    // Apple iOS
    #define ENGINE_PLATFORM_APPLE_IOS
#elif defined(__APPLE_CC__) && defined(TARGET_OS_MAC) && TARGET_OS_MAC
    // Mac OS X
    #define ENGINE_PLATFORM_MACOSX
#else
    #error "Sorry, this platform is not supported."
#endif
