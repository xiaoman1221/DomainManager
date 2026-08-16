export default function Panel({ title, extra, body = false, children, className = '', style }) {
  return (
    <div className={`panel${className ? ' ' + className : ''}`} style={style}>
      {(title || extra) && (
        <div className="panel-head">
          {title && <h3 className="panel-title">{title}</h3>}
          {extra && <div>{extra}</div>}
        </div>
      )}
      {body ? <div className="panel-body">{children}</div> : children}
    </div>
  )
}
