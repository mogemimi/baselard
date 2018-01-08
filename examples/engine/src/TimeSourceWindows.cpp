#include <engine/TimeSourceWindows.h>
#include <engine/PrerequisitesWindows.h>
#include <assert>

namespace Engine {

TimeSourceWindows::TimeSourceWindows()
{
    LARGE_INTEGER frequency;
    ::QueryPerformanceFrequency(&frequency);

#ifdef DEBUG
    assert(0 != frequency.QuadPart);
#endif

    secondsPerTick = 1.0 / frequency.QuadPart;
}

TimePoint TimeSourceWindows::Now() const
{
    LARGE_INTEGER time;
    ::QueryPerformanceCounter(&time);
    auto currentSeconds = time.QuadPart * secondsPerTick;
    return TimePoint(Duration(currentSeconds));
}

} // namespace Engine
