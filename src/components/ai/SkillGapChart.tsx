import {
  Legend,
  PolarAngleAxis,
  PolarGrid,
  PolarRadiusAxis,
  Radar,
  RadarChart,
  ResponsiveContainer,
  Tooltip,
} from "recharts";

import type { SkillGap } from "@/types";

export function SkillGapChart({ gaps }: { gaps: SkillGap[] }) {
  return (
    <div className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <RadarChart data={gaps} outerRadius="72%">
          <PolarGrid stroke="var(--border)" />
          <PolarAngleAxis dataKey="skill" tick={{ fontSize: 11, fill: "var(--muted-foreground)" }} />
          <PolarRadiusAxis domain={[0, 100]} tick={false} axisLine={false} />
          <Tooltip
            contentStyle={{
              background: "var(--popover)",
              border: "1px solid var(--border)",
              borderRadius: 10,
              fontSize: 12,
              color: "var(--popover-foreground)",
            }}
          />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <Radar
            name="Role requirement"
            dataKey="required"
            stroke="var(--chart-2)"
            fill="var(--chart-2)"
            fillOpacity={0.18}
          />
          <Radar
            name="Your level"
            dataKey="current"
            stroke="var(--chart-1)"
            fill="var(--chart-1)"
            fillOpacity={0.3}
          />
        </RadarChart>
      </ResponsiveContainer>
    </div>
  );
}
