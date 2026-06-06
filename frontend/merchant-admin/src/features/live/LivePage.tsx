import { StateBlock } from '../../components/common/StateBlock'

export function LivePage() {
  return (
    <>
      <div className="page-header">
        <div>
          <h1>直播管理</h1>
          <p>直播间列表、推流信息、开播和结束直播入口会放在这里。</p>
        </div>
        <span className="status-tag muted">占位页</span>
      </div>

      <section className="panel placeholder-panel">
        <StateBlock
          title="直播管理暂未接入"
          description="当前先保留导航和空页面，后续可对接 /api/live/rooms、开播、结束直播和推流信息接口。"
        />
      </section>
    </>
  )
}
