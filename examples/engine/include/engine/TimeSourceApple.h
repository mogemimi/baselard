#pragma once

#include <engine/TimePoint.h>

namespace Engine {

class TimeSourceApple {
public:
    TimeSourceApple();

    TimePoint Now() const;

private:
    double secondsPerTick;
};

} // namespace Engine
