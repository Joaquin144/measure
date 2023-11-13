"use client"

import React, { useState, useEffect } from 'react';
import { ResponsiveSankey } from '@nivo/sankey'

interface UserFlowProps {
  authToken: string,
  appId:string, 
  startDate:string,
  endDate:string,
  appVersion:string,
}

async function getUserFlowData( authToken: string, appId:string, startDate:string, endDate:string, appVersion:string) {
  const origin = "https://frosty-fog-7165.fly.dev"
  const opts = {
    headers: {
      "Authorization": `Bearer ${authToken}`
    }
  };

  const serverFormattedStartDate = new Date(startDate).toISOString()
  const serverFormattedEndDate = new Date(endDate).toISOString()
  const fakeUUID = 'e2d2f609-7425-4077-a7ff-1d09e62c84d6'
  return await fetch(`${origin}/apps/${fakeUUID}/journey?version=${appVersion}&from=${serverFormattedStartDate}&to=${serverFormattedEndDate}`, opts);
}

const emptyData = {
  "nodes": [
    {
      "id": "",
      "nodeColor": "",
      "issues": {
        "crashes": [],
        "anrs":[]
      }
    },
  ],
  "links": [
  ]
}

const formatter = Intl.NumberFormat('en', { notation: 'compact' });

const UserFlow: React.FC<UserFlowProps> = ({ authToken, appId, startDate, endDate, appVersion }) => {
    const [data, setData] = useState(emptyData);
    const [message, setMessage] = useState("");

    const getData = async (authToken:string, appId:string, startDate:string, endDate:string, appVersion:string) => {
      setMessage("Updating data...")
      const res = await getUserFlowData(authToken, appId, startDate, endDate, appVersion)
      if(!res.ok) {
        setMessage("Error fetching data! Please try again")
      } else {
        setMessage("")
        setData(await res.json())
      }
    }
    
    useEffect(() => {
      getData(authToken, appId, startDate, endDate, appVersion)
    }, [authToken, appId, startDate, endDate, appVersion]);

    return (
      <div className="flex items-center justify-center border border-black text-black font-sans text-sm w-5/6 h-screen">
        { message!=="" && <p className="text-lg">{message}</p> }
        { message === "" && <ResponsiveSankey
                              data={data}
                              margin={{ top: 80, right: 120, bottom: 80, left: 120 }}
                              align="justify"
                              colors={({nodeColor}) => nodeColor}
                              nodeOpacity={1}
                              nodeHoverOthersOpacity={0.35}
                              nodeThickness={18}
                              nodeSpacing={24}
                              nodeBorderWidth={0}
                              nodeBorderColor={{
                                  from: 'color',
                                  modifiers: [
                                      [
                                          'darker',
                                          0.8
                                      ]
                                  ]
                              }}
                              nodeBorderRadius={3}
                              linkOpacity={0.25}
                              linkHoverOthersOpacity={0.1}
                              linkContract={3}
                              enableLinkGradient={false}
                              labelPosition="outside"
                              labelOrientation="horizontal"
                              labelPadding={16}
                              labelTextColor="#000000"
                              nodeTooltip={({
                                node
                              }) => <div className="pointer-events-none z-50 rounded-md p-4 bg-neutral-800">
                                       <p className="font-sans text-white">{node.label}</p>
                                       {node.issues.crashes.length > 0 && 
                                        <div>
                                          <div className="py-2"/>
                                          <p className="font-sans text-white">Crashes:</p>
                                          <ul className="list-disc">
                                              {node.issues.crashes.map(({ title, count }) => (
                                                  <li key={title}>
                                                      <span className="font-sans text-white text-xs">{title} - {formatter.format(count)}</span>
                                                  </li>
                                              ))}
                                          </ul>
                                        </div>
                                       }
                                       {node.issues.anrs.length > 0 && 
                                        <div>
                                          <div className="py-2"/>
                                          <p className="font-sans text-white">ANRs:</p>
                                          <ul className="list-disc">
                                              {node.issues.anrs.map(({ title, count }) => (
                                                  <li key={title}>
                                                      <span className="font-sans text-white text-xs">{title} - {formatter.format(count)}</span>
                                                  </li>
                                              ))}
                                          </ul>
                                        </div>
                                       }
                                    </div>}
                              linkTooltip={({
                                link
                              }) => <div className="pointer-events-none z-50 rounded-md p-4 bg-neutral-800">
                                       <p className="font-sans text-white">{link.source.label} &gt; {link.target.label} - {formatter.format(link.value)} </p>
                                    </div>}
                            />}
      </div>
  );
};

export default UserFlow;