#include <engine/engine.h>
#include <engine/Platform.h>

#if defined(ENGINE_PLATFORM_LINUX)
    #include <engine/TimeSourceLinux.h>
#elif defined(ENGINE_PLATFORM_WINDOWS)
    #include <engine/TimeSourceWindows.h>
#elif defined(ENGINE_PLATFORM_MACOSX)
    #include <engine/TimeSourceApple.h>
#endif

extern "C" {
#include "stringify.h"
}

#include <iostream>

namespace Engine {

#if defined(ENGINE_PLATFORM_LINUX)
using TimeSource = TimeSourceLinux;
#elif defined(ENGINE_PLATFORM_WINDOWS)
using TimeSource = TimeSourceWindows;
#elif defined(ENGINE_PLATFORM_MACOSX)
using TimeSource = TimeSourceApple;
#endif

void Engine::Run()
{
    char buf[100];
    stringify_int(42, buf, 100);
    std::cout << "stringify_int = " << buf << std::endl;

#if defined(ENGINE_PLATFORM_LINUX) || \
    defined(ENGINE_PLATFORM_WINDOWS) || \
    defined(ENGINE_PLATFORM_MACOSX)
    TimeSource timeSource;
    auto now = timeSource.Now();
    auto duration = std::chrono::duration_cast<Duration>(now.time_since_epoch());
    std::cout << "now = " << duration.count() << std::endl;
#else
    std::cout << "hi" << std::endl;
#endif
}

} // namespace Engine
