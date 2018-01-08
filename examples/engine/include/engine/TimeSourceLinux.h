#pragma once

#include <engine/TimePoint.h>

namespace Engine {

class TimeSourceLinux {
public:
    TimeSourceLinux() = default;

    TimePoint Now() const;
};

} // namespace Engine
