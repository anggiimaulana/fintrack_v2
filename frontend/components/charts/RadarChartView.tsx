"use client";

import { Radar, RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, ResponsiveContainer, Tooltip } from "recharts";

const data = [
  { subject: "Makanan", A: 120, fullMark: 150 },
  { subject: "Transport", A: 98, fullMark: 150 },
  { subject: "Utilitas", A: 86, fullMark: 150 },
  { subject: "Hiburan", A: 99, fullMark: 150 },
  { subject: "Kesehatan", A: 85, fullMark: 150 },
  { subject: "Pendidikan", A: 65, fullMark: 150 },
];

export function RadarChartView() {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <RadarChart cx="50%" cy="50%" outerRadius="80%" data={data} margin={{ top: 20, right: 30, bottom: 20, left: 30 }}>
        <PolarGrid stroke="#e2e8f0" />
        <PolarAngleAxis dataKey="subject" tick={{ fill: '#475569', fontSize: 12 }} />
        <PolarRadiusAxis angle={30} domain={[0, 150]} tick={false} axisLine={false} />
        <Radar name="Pengeluaran" dataKey="A" stroke="#f43f5e" fill="#f43f5e" fillOpacity={0.5} />
        <Tooltip 
          contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.08)' }}
        />
      </RadarChart>
    </ResponsiveContainer>
  );
}
