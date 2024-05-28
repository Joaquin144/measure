"use client"

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { AppVersion, JourneyApiStatus, emptyJourney, fetchJourneyFromServer } from '../api/api_calls';
import Dagre from '@dagrejs/dagre';
import ReactFlow, {
  useNodesState,
  useEdgesState,
  Position,
  Handle,
} from 'reactflow';
import 'reactflow/dist/style.css';
import Link from 'next/link';

interface JourneyProps {
  teamId: string,
  appId: string,
  startDate: string,
  endDate: string,
  appVersion: AppVersion,
}

type Node = {
  id: string
  type: string | undefined
  width: number
  height: number
  position: { x: number, y: number }
  sourcePosition: Position | undefined
  targetPosition: Position | undefined
  data: {
    label: string,
    teamId: string,
    appId: string,
    startDate: string,
    endDate: string,
    issues: {
      crashes: { id: string, title: string; count: number }[];
      anrs: { id: string, title: string; count: number }[];
    },
    nodeIssueContribution: number
  }
}

type Edge = {
  id: string
  source: string
  target: string
  sourceHandle: string
  style: {
    strokeWidth: number
    stroke: string
  }
  label: string
  animated: boolean
}

const nodeTypes = {
  measureNode: MeasureNode,
};

const formatter = Intl.NumberFormat('en', { notation: 'compact' });

{/* @ts-ignore */ }
function MeasureNode({ data, isConnectable }) {

  const isNodeWithIssues = data.issues.crashes.length > 0 || data.issues.anrs.length > 0
  return (
    <div className='group border-black rounded-md transition ease-in-out duration-300 hover:-translate-y-1 hover:scale-110'>
      <Handle type="target" id="a" position={Position.Top} isConnectable={isConnectable} />
      <Handle type="source" id="b" position={Position.Bottom} isConnectable={isConnectable} />

      {/* this div is a hack to animate label position from center to left and back again on hover */}
      <div className={`w-full flex flex-row p-4 ${isNodeWithIssues ? 'bg-red-400' : 'bg-emerald-400'}`}>
        <div className="grow group-hover:grow-0 transition-[flex-grow] ease-out duration-300" />
        <p className="font-sans text-white w-fit">{data.label}</p>
        <div className="grow group-hover:grow-0 transition-[flex-grow] ease-out duration-300" />
      </div>

      {/* Indicator line to show percentage contribution of issues */}
      {isNodeWithIssues &&
        <div className='w-full bg-red-400'>
          <div className={`h-1 bg-neutral-950`} style={{ width: `${data.nodeIssueContribution * 100}%` }} />
        </div>}

      <div className='h-0 rounded-b-md opacity-0 bg-neutral-950 group-hover:pl-2 group-hover:pr-2 group-hover:opacity-100 group-hover:h-full transition ease-in-out duration-300 '>
        {data.issues.crashes.length > 0 && (
          <div>
            <p className="font-sans text-white pt-2">Crashes:</p>
            <ul className="list-disc list-inside text-white pt-2 pb-4 pl-2 pr-2">
              {/* @ts-ignore */}
              {data.issues.crashes.map(({ id, title, count }) => (
                <li key={title}>
                  <span className="font-sans text-xs">
                    <Link href={`/${data.teamId}/crashes/${data.appId}/${id}/${title}?start_date=${data.startDate}&end_date=${data.endDate}`} className="underline decoration-yellow-200 hover:decoration-yellow-500">
                      {title} - {formatter.format(count)}
                    </Link>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
        {data.issues.anrs.length > 0 && (
          <div>
            <p className="font-sans text-white pt-2">ANRs:</p>
            <ul className="list-disc list-inside text-white pt-2 pb-4 pl-2 pr-2">
              {/* @ts-ignore */}
              {data.issues.anrs.map(({ id, title, count }) => (
                <li key={title}>
                  <span className="font-sans text-xs">
                    <Link href={`/${data.teamId}/anrs/${data.appId}/${id}/${title}?start_date=${data.startDate}&end_date=${data.endDate}`} className="underline decoration-yellow-200 hover:decoration-yellow-500">
                      {title} - {formatter.format(count)}
                    </Link>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div >
  );
};

const getReactFlowFromJourney = (teamId: string, appId: string, startDate: string, endDate: string, journey: typeof emptyJourney) => {
  // Return null flow if no nodes
  if (journey.nodes === null) {
    return { nodes: [], edges: [] }
  }

  const nodes = journey.nodes.map(node => {
    const { id, issues } = node;

    let nodeTotalIssueCount = 0
    node.issues.crashes.map((crash) => {
      nodeTotalIssueCount += crash.count
    })

    node.issues.anrs.map((anr) => {
      nodeTotalIssueCount += anr.count
    })

    const nodeIssueContribution = nodeTotalIssueCount / journey.totalIssues

    return {
      id,
      position: { x: 0, y: 0 },
      width: 500,
      height: 200,
      type: 'measureNode',
      sourcePosition: undefined,
      targetPosition: undefined,
      data: {
        label: id,
        teamId,
        appId,
        startDate,
        endDate,
        issues: issues, nodeIssueContribution
      }
    };
  });

  // Return flow with only nodes if no links
  if (journey.links === null) {
    return { nodes, edges: [] }
  }

  const edges = journey.links.map(link => {
    const { source, target, value } = link;

    return {
      id: source + '-' + target,
      source,
      target,
      sourceHandle: 'b',
      label: formatter.format(value) + ' sessions',
      animated: true
    };
  });

  return {
    nodes,
    edges
  };
}

const dagreGraph = new Dagre.graphlib.Graph();
dagreGraph.setDefaultEdgeLabel(() => ({}));

const getLayoutedElements = (nodes: Node[], edges: Edge[]) => {
  dagreGraph.setGraph({ rankdir: 'TB', align: 'DR', nodesep: 200, edgesep: 400 });

  nodes.forEach((node) => dagreGraph.setNode(node.id, node));
  edges.forEach((edge) => dagreGraph.setEdge(edge.source, edge.target));

  Dagre.layout(dagreGraph);

  nodes.forEach((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    node.position = {
      x: nodeWithPosition.x,
      y: nodeWithPosition.y
    };
    return node;
  });

  return { nodes, edges };
};

const Journey: React.FC<JourneyProps> = ({ teamId, appId, startDate, endDate, appVersion }) => {

  const [journeyApiStatus, setJourneyApiStatus] = useState(JourneyApiStatus.Loading);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const router = useRouter()

  const getJourney = async (appId: string, startDate: string, endDate: string, appVersion: AppVersion) => {
    setJourneyApiStatus(JourneyApiStatus.Loading)

    const result = await fetchJourneyFromServer(appId, startDate, endDate, appVersion, router)

    switch (result.status) {
      case JourneyApiStatus.Error:
        setJourneyApiStatus(JourneyApiStatus.Error)
        break
      case JourneyApiStatus.Success:
        setJourneyApiStatus(JourneyApiStatus.Success)
        let flow = getReactFlowFromJourney(teamId, appId, startDate, endDate, result.data)

        if (flow.nodes.length === 0) {
          setJourneyApiStatus(JourneyApiStatus.NoData)
          break
        }

        const layoutedFlow = getLayoutedElements(flow.nodes as Node[], flow.edges as Edge[]);
        setNodes(layoutedFlow.nodes)
        setEdges(layoutedFlow.edges)
        break
    }
  }

  useEffect(() => {
    getJourney(appId, startDate, endDate, appVersion)
  }, [appId, startDate, endDate, appVersion]);

  return (
    <div className="flex items-center justify-center border border-black text-black font-sans text-sm w-5/6 h-[600px]">
      {journeyApiStatus === JourneyApiStatus.Loading && <p className="text-lg">Updating journey...</p>}
      {journeyApiStatus === JourneyApiStatus.Error && <p className="text-lg">Error fetching journey. Please refresh page or change filters to try again.</p>}
      {journeyApiStatus === JourneyApiStatus.NoData && <p className="text-lg">No data</p>}
      {journeyApiStatus === JourneyApiStatus.Success
        && <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodesDraggable={true}
          nodesConnectable={false}
          edgesUpdatable={false}
          proOptions={{ hideAttribution: true }}
          // @ts-ignore
          nodeTypes={nodeTypes}
          fitView
        />}
    </div>
  );
};

export default Journey;