"use client"

import React, { useEffect, useState } from 'react';
import { ResponsiveLine } from '@nivo/line'
import { ExceptionsDetailsPlotApiStatus, ExceptionsType, fetchExceptionsDetailsPlotFromServer } from '../api/api_calls';
import { useRouter } from 'next/navigation';
import { formatDateToHumanReadableDate } from '../utils/time_utils';
import { Filters } from './filters';

interface ExceptionsDetailsPlotProps {
  exceptionsType: ExceptionsType,
  exceptionsGroupId: string,
  filters: Filters
}

type ExceptionsDetailsPlot = {
  id: string
  data: {
    x: string
    y: number
  }[]
}[]

const ExceptionsDetailsPlot: React.FC<ExceptionsDetailsPlotProps> = ({ exceptionsType, exceptionsGroupId, filters }) => {
  const router = useRouter()

  const [exceptionsDetailsPlotApiStatus, setExceptionsDetailsPlotApiStatus] = useState(ExceptionsDetailsPlotApiStatus.Loading);
  const [plot, setPlot] = useState<ExceptionsDetailsPlot>();

  const getExceptionsDetailsPlot = async () => {
    // Don't try to fetch plot if filters aren't ready
    if (!filters.ready) {
      return
    }

    setExceptionsDetailsPlotApiStatus(ExceptionsDetailsPlotApiStatus.Loading)

    const result = await fetchExceptionsDetailsPlotFromServer(exceptionsType, exceptionsGroupId, filters, router)

    switch (result.status) {
      case ExceptionsDetailsPlotApiStatus.Error:
        setExceptionsDetailsPlotApiStatus(ExceptionsDetailsPlotApiStatus.Error)
        break
      case ExceptionsDetailsPlotApiStatus.NoData:
        setExceptionsDetailsPlotApiStatus(ExceptionsDetailsPlotApiStatus.NoData)
        break
      case ExceptionsDetailsPlotApiStatus.Success:
        setExceptionsDetailsPlotApiStatus(ExceptionsDetailsPlotApiStatus.Success)

        // map result data to chart format
        setPlot(result.data.map((item: any) => ({
          id: item.id,
          data: item.data.map((data: any) => ({
            x: data.datetime,
            y: data.instances,
          })),
        })))
        break
    }
  }

  useEffect(() => {
    getExceptionsDetailsPlot()
  }, [exceptionsType, exceptionsGroupId, filters]);

  return (
    <div className="flex border border-black font-sans items-center justify-center w-full h-[32rem]">
      {exceptionsDetailsPlotApiStatus === ExceptionsDetailsPlotApiStatus.Error && <p className="text-lg font-display text-center p-4">Error fetching plot, please change filters or refresh page to try again</p>}
      {exceptionsDetailsPlotApiStatus === ExceptionsDetailsPlotApiStatus.NoData && <p className="text-lg font-display text-center p-4">No Data</p>}
      {exceptionsDetailsPlotApiStatus === ExceptionsDetailsPlotApiStatus.Success &&
        <ResponsiveLine
          data={plot!}
          curve="monotoneX"
          colors={{ scheme: 'nivo' }}
          margin={{ top: 40, right: 160, bottom: 120, left: 120 }}
          xFormat="time:%Y-%m-%d"
          xScale={{
            format: '%Y-%m-%d',
            precision: 'day',
            type: 'time',
            useUTC: false
          }}
          yScale={{
            type: 'linear',
            min: 'auto',
            max: 'auto'
          }}
          yFormat=" >-.2f"
          axisTop={null}
          axisRight={null}
          axisBottom={{
            legend: 'Date',
            tickPadding: 10,
            legendOffset: 100,
            format: '%b %d, %Y',
            tickRotation: 45,
            legendPosition: 'middle'
          }}
          axisLeft={{
            tickSize: 1,
            tickPadding: 5,
            format: value => Number.isInteger(value) ? value : '',
            legend: exceptionsType === ExceptionsType.Crash ? 'Crash instances' : 'ANR instances',
            legendOffset: -80,
            legendPosition: 'middle'
          }}
          pointSize={3}
          pointBorderWidth={2}
          pointBorderColor={{
            from: 'color',
            modifiers: [
              [
                'darker',
                0.3
              ]
            ]
          }}
          pointLabelYOffset={-12}
          useMesh={true}
          legends={[
            {
              anchor: 'bottom-right',
              direction: 'column',
              justify: false,
              translateX: 100,
              translateY: 0,
              itemsSpacing: 0,
              itemDirection: 'left-to-right',
              itemWidth: 80,
              itemHeight: 20,
              itemOpacity: 0.75,
              symbolSize: 12,
              symbolShape: 'circle',
              symbolBorderColor: 'rgba(0, 0, 0, .5)',
            }
          ]}
          enableSlices="x"
          sliceTooltip={({ slice }) => {
            return (
              <div className="bg-neutral-950 text-white flex flex-col p-2 text-xs">
                <p className='p-2'>Date: {formatDateToHumanReadableDate(slice.points[0].data.xFormatted.toString())}</p>
                {slice.points.map((point) => (
                  <div className="flex flex-row items-center p-2" key={point.id}>
                    <div className="w-2 h-2 rounded-full" style={{ backgroundColor: point.serieColor }} />
                    <div className="px-2" />
                    <p>{point.data.y.toString()} {point.data.y.valueOf() as number > 1 ? 'instances' : 'instance'}</p>
                  </div>
                ))}
              </div>
            )
          }}
        />}
    </div>
  )

};

export default ExceptionsDetailsPlot;