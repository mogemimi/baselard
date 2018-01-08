#include <engine/engine.h>
#include <engine/Platform.h>

#if defined(ENGINE_PLATFORM_LINUX)
    #include <engine/TimeSourceLinux.h>
#elif defined(ENGINE_PLATFORM_WINDOWS)
    #include <engine/TimeSourceWindows.h>
#elif defined(ENGINE_PLATFORM_MACOSX)
    #include <engine/TimeSourceApple.h>
#else
    #error "Sorry, this platform is not supported."
#endif

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
    TimeSource timeSource;

    auto now = timeSource.Now();
    auto duration = std::chrono::duration_cast<Duration>(now.time_since_epoch());

    std::cout << "now = " << duration.count() << std::endl;
}

} // namespace Engine
